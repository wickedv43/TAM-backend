package server

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/wickedv43/TAM-backend/internal/common"
	"github.com/wickedv43/TAM-backend/internal/database/ent"
)

const (
	minWithdrawTON = 0.01
	maxWithdrawTON = 10000
	withdrawFeeTON = 0.01
)

// Withdraw sends TON from the user's subwallet (wallet_hd_id) to the given address.
// Balance in DB is deducted first; on send failure it is refunded.
//
// @Summary      Withdraw TON
// @Description  Send TON from your subwallet to the given address. Amount is deducted from your balance.
// @Tags         customers
// @Accept       json
// @Produce      json
// @Param        input body common.WithdrawRequest true "Amount in TON and destination address"
// @Success      202  {object}  common.WithdrawResponse
// @Failure      400  {object}  string "Invalid request / wallet not linked / invalid address"
// @Failure      401  {object}  string "Unauthorized"
// @Failure      403  {object}  string "Insufficient balance"
// @Failure      500  {object}  string "Internal error"
// @Security     BearerAuth
// @Router       /customers/withdraw [post]
func (s *Server) Withdraw(c echo.Context) error {
	customerID, err := s.GetContextUser(c)
	if err != nil {
		s.log.Errorf("getting customer ID: %s", err)
		status := common.GetHTTPStatus(common.ErrUnauthorized)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrUnauthorized)})
	}

	var req common.WithdrawRequest
	if err = c.Bind(&req); err != nil {
		s.log.Errorf("binding request: %s", err)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}

	if req.Amount < minWithdrawTON {
		s.log.Errorf("insufficient amount of %s to withdraw", req.Amount)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": "Minimum withdrawal is 0.01 TON"})
	}
	if req.Amount > maxWithdrawTON {
		s.log.Errorf("insufficient amount of %s to withdraw", req.Amount)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": "Maximum withdrawal is 10000 TON"})
	}
	toAddress := strings.TrimSpace(req.ToAddress)
	if toAddress == "" {
		s.log.Errorf("empty address")
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": "to_address is required"})
	}

	ctx := c.Request().Context()
	cust, err := s.customers.GetCustomer(ctx, customerID)
	if err != nil {
		if ent.IsNotFound(err) {
			s.log.Errorf("customer %s not found", customerID)
			status := common.GetHTTPStatus(common.ErrNotFound)
			return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrNotFound)})
		}
		s.log.Errorf("withdraw: get customer: %v", err)
		status := common.GetHTTPStatus(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInternalError)})
	}

	// Send from user's subwallet (wallet_hd_id); must be assigned
	if cust.WalletHdID < 0 {
		s.log.Errorf("customer %s has invalid address", customerID)
		status := common.GetHTTPStatus(common.ErrWalletNotLinked)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrWalletNotLinked)})
	}
	amountWithFee := req.Amount + withdrawFeeTON

	if amountWithFee > cust.TonBalance {
		s.log.Errorf("insufficient amount of %s to withdraw", req.Amount)
		status := common.GetHTTPStatus(common.ErrInsufficientBalance)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInsufficientBalance)})
	}

	if err = s.customers.DeductBalance(ctx, customerID, amountWithFee); err != nil {
		if errors.Is(err, common.ErrInsufficientBalance) {
			s.log.Errorf("insufficient amount of %s to withdraw", req.Amount)
			status := common.GetHTTPStatus(common.ErrInsufficientBalance)
			return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInsufficientBalance)})
		}
		s.log.Errorf("withdraw: deduct balance: %v", err)
		status := common.GetHTTPStatus(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInternalError)})
	}

	// Send from user's subwallet; fee is taken from amount (mode 0). Bounded wait to avoid infinite retries.
	txHash, err := s.ton.SendTON(ctx, toAddress, req.Amount, "Withdrawal", uint32(cust.WalletHdID))
	if err != nil {
		s.log.Errorf("withdraw: send TON: %v", err)
		if refErr := s.customers.RefundBalance(ctx, customerID, amountWithFee); refErr != nil {
			s.log.Errorf("withdraw: refund after send failure: %v", refErr)
		}
		status := common.GetHTTPStatus(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInternalError)})
	}

	return c.JSON(http.StatusAccepted, common.WithdrawResponse{
		TxHash: base64.StdEncoding.EncodeToString(txHash),
	})
}
