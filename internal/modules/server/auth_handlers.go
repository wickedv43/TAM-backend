package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/wickedv43/TAM-backend/internal/common"
	"github.com/wickedv43/TAM-backend/internal/common/constants/customer_status"
	"github.com/wickedv43/TAM-backend/internal/database/ent"

	initdata "github.com/telegram-mini-apps/init-data-golang"
)

// @TODO: post-mvp: allow to update customer state on every auth handler request?
// onAuth handles user authentication
// @Summary      Authenticate user
// @Description  Authenticates user via Telegram init data
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        input body common.AuthRequest true "Auth request"
// @Success      200  {string}  string "ok"
// @Failure      400  {object}  string "Invalid request"
// @Failure      401  {object}  string "Unauthorized"
// @Failure      500  {object}  string "Internal error"
// @Router       /auth [post]
func (s *Server) onAuth(c echo.Context) error {
	var (
		authReq common.AuthRequest
		token   = s.cfg.Bot.Token
		expIn   = s.cfg.InitDataExpire
	)

	err := c.Bind(&authReq)
	if err != nil {
		s.log.Warnf("binding auth request: %v", err)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}

	//validate and parse init data
	if validateErr := initdata.Validate(authReq.InitData, token, expIn); validateErr != nil {
		s.log.Errorf("validate init_data: %v", validateErr)
		status := common.GetHTTPStatus(common.ErrUnauthorized)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrUnauthorized)})
	}

	cData, err := initdata.Parse(authReq.InitData)
	if err != nil {
		s.log.Errorf("parsing init_data: %v", err)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}
	s.log.Debugf("init_data: %v", cData)

	//check customer
	existingCustomer, err := s.customers.GetCustomerByTgID(c.Request().Context(), cData.User.ID)
	if err != nil {
		if !ent.IsNotFound(err) {
			s.log.Errorf("Error while checking if customer exists: %v", err)
			status := common.GetHTTPStatus(common.ErrInternalError)
			return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInternalError)})
		}

		//create new user pipe
		customer := &ent.Customer{
			TgID:        cData.User.ID,
			TgUsername:  cData.User.Username,
			TgFirstname: cData.User.FirstName,
			TgLastname:  cData.User.LastName,
			TgLanguage:  cData.User.LanguageCode,
			TgPicture:   cData.User.PhotoURL,
			TgIsPremium: cData.User.IsPremium,
			TgIsAllowPm: cData.User.AllowsWriteToPm,
			Status:      int(customer_status.REGISTERED),
		}

		existingCustomer, err = s.customers.CreateCustomer(c.Request().Context(), customer)
		if err != nil {
			s.log.Errorf("Error creating customer: %v", err)
			status := common.GetHTTPStatus(common.ErrInternalError)
			return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInternalError)})
		}

		s.log.Infof("auth: existingCustomer.WalletHdID=%d", existingCustomer.WalletHdID)

		var (
			addrBounceable, addrNonBounceable string
		)
		addrBounceable, addrNonBounceable, err = s.ton.GetAddress(uint32(existingCustomer.WalletHdID))
		if err != nil {
			s.log.Errorf("Error getting address: %v", err)
			status := common.GetHTTPStatus(common.ErrInternalError)
			return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInternalError)})
		}

		existingCustomer, err = s.customers.UpdateCustomer(c.Request().Context(), existingCustomer.ID, func(u *ent.CustomerUpdateOne) {
			u.SetAddressBounceable(addrBounceable)
			u.SetAddressNonbounceable(addrNonBounceable)
		})

		if err != nil {
			s.log.Errorf("Error updating customer addresses: %v", err)
			status := common.GetHTTPStatus(common.ErrInternalError)
			return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInternalError)})
		}
	}

	if existingCustomer.Status == int(customer_status.PRE_REGISTRATION) {
		existingCustomer, err = s.customers.UpdateCustomer(c.Request().Context(), existingCustomer.ID, func(u *ent.CustomerUpdateOne) {
			u.SetTgUsername(cData.User.Username)
			u.SetTgFirstname(cData.User.FirstName)
			u.SetTgLastname(cData.User.LastName)
			u.SetTgLanguage(cData.User.LanguageCode)
			u.SetTgPicture(cData.User.PhotoURL)
			u.SetTgIsPremium(cData.User.IsPremium)
			u.SetTgIsAllowPm(cData.User.AllowsWriteToPm)
			u.SetStatus(int(customer_status.REGISTERED))
		})

		if err != nil {
			s.log.Errorf("Error updating customer state: %v", err)
			status := common.GetHTTPStatus(common.ErrInternalError)
			return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInternalError)})
		}
	}

	jwtToken, err := s.generateJWT(existingCustomer)
	if err != nil {
		s.log.Errorf("generate jwt: %v", err)
		status := common.GetHTTPStatus(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInternalError)})
	}

	c.Response().Header().Set("Authorization", "Bearer "+jwtToken)

	return c.JSON(http.StatusOK, "ok")
}
