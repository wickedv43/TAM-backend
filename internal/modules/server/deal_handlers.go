package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/wickedv43/TAM-backend/internal/common"
	"github.com/wickedv43/TAM-backend/internal/common/constants/customer_channel_role"
	"github.com/wickedv43/TAM-backend/internal/common/constants/deal_status"
	"github.com/wickedv43/TAM-backend/internal/common/constants/deal_target_type"
	"github.com/wickedv43/TAM-backend/internal/common/constants/deal_type"
	"github.com/wickedv43/TAM-backend/internal/database/ent"
	"github.com/wickedv43/TAM-backend/internal/messages"
	"github.com/wickedv43/TAM-backend/internal/modules/telegram_bot/util"
)

// OfferDeal creates a new deal
// @Summary      Offer deal
// @Description  Create a new advertising offer
// @Tags         deals
// @Accept       json
// @Produce      json
// @Param        input body common.OfferDealRequest true "Offer details"
// @Success      201  {object}  common.DealResponse
// @Failure      400  {object}  string "Invalid request"
// @Failure      401  {object}  string "Unauthorized"
// @Failure      404  {object}  string "Channel not found"
// @Failure      500  {object}  string "Internal error"
// @Security     BearerAuth
// @Router       /deals/offer-deal [post]
func (s *Server) OfferDeal(c echo.Context) error {
	customerID, err := s.GetContextUser(c)
	if err != nil {
		s.log.Errorf("invalid user id: %v", err)
		status := common.GetHTTPStatus(common.ErrUnauthorized)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrUnauthorized)})
	}

	var req common.OfferDealRequest
	if err = c.Bind(&req); err != nil {
		s.log.Errorf("invalid request body: %v", err)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}
	ctx := c.Request().Context()

	channel, err := s.channels.GetChannel(ctx, req.ChannelID)
	if err != nil {
		if ent.IsNotFound(err) {
			s.log.Errorf("channel %s not found", req.ChannelID)
			status := common.GetHTTPStatus(common.ErrChannelNotFound)
			return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrChannelNotFound)})
		}
		s.log.Errorf("get channel: %v", err)
		status := common.GetHTTPStatus(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInternalError)})
	}

	channelCustomers, err := s.channels.GetChannelCustomers(ctx, req.ChannelID)
	if err != nil {
		s.log.Errorf("get channel customers: %v", err)
		status := common.GetHTTPStatus(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInternalError)})
	}

	for _, cc := range channelCustomers {
		if cc.CustomerID == customerID {
			s.log.Errorf("offer deal forbidden for channel %s by its customer %s", req.ChannelID, customerID)
			status := common.GetHTTPStatus(common.ErrForbidden)
			return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrForbidden)})
		}
	}

	var channelManagerID uuid.UUID
	for _, cc := range channelCustomers {
		if cc.Role == int(customer_channel_role.ADMIN) ||
			cc.Role == int(customer_channel_role.OWNER) {
			channelManagerID = cc.CustomerID
			break
		}
	}

	if channelManagerID == uuid.Nil {
		s.log.Errorf("channel customer manager not found")
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}

	duration := common.GetDealStatusDuration(deal_status.DRAFT)
	now := time.Now()

	// Validate target_type (required)
	if req.TargetType == "" {
		s.log.Errorf("target type is required")
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}
	if !deal_target_type.DealTargetType(req.TargetType).IsValid() {
		s.log.Errorf("invalid deal target type: %s", req.TargetType)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}
	targetType := req.TargetType

	var tonPrice float64
	prices := common.ChannelPricesFromMap(channel.Prices)
	if prices != nil {
		tonPrice = prices.GetPrice(targetType)
	}
	if tonPrice <= 0 {
		s.log.Errorf("get channel customer channel price: %v", targetType)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}

	deal := &ent.Deal{
		ChannelID:            req.ChannelID,
		AdvertiserCustomerID: customerID,
		ChannelManagerID:     channelManagerID,
		Type:                 int(deal_type.ADVERTISING_OFFER),
		Status:               int(deal_status.DRAFT),
		StatusUpdatedAt:      now,
		ExpiresAt:            now.Add(duration),
		TargetType:           targetType,
		TonPrice:             tonPrice,
	}

	createdDeal, err := s.deals.CreateDeal(ctx, deal)
	if err != nil {
		s.log.Errorf("create deal: %v", err)
		status := common.GetHTTPStatus(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInternalError)})
	}

	return c.JSON(http.StatusCreated, common.MapDeal(createdDeal))
}

// SendDealData
// @Summary      Send deal data
// @Description  Submit post text or content for a deal
// @Tags         deals
// @Accept       json
// @Produce      json
// @Param        input body common.SendDealDataRequest true "Deal data"
// @Success      202  {string}  string "ok"
// @Failure      400  {object}  string "Invalid request"
// @Failure      403  {object}  string "Forbidden"
// @Failure      404  {object}  string "Deal not found"
// @Security     BearerAuth
// @Router       /deals/send-deal-data [post]
func (s *Server) SendDealData(c echo.Context) error {
	var req common.SendDealDataRequest

	if err := c.Bind(&req); err != nil {
		s.log.Errorf("bind request body: %v", err)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		message := s.GetUserMessage(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": message})
	}

	if req.DealID == uuid.Nil {
		s.log.Errorf("deal id is required")
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}

	deal := c.Get(ctxDeal).(*ent.Deal)

	if deal.Status != int(deal_status.DRAFT) {
		s.log.Errorf("send-deal-data: deal %s status %d is not DRAFT", deal.ID, deal.Status)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}

	customerID, err := s.GetContextUser(c)
	if err != nil {
		s.log.Errorf("get customer id: %v", err)
		status := common.GetHTTPStatus(common.ErrUnauthorized)
		message := s.GetUserMessage(common.ErrUnauthorized)
		return c.JSON(status, map[string]string{"error": message})
	}

	var botReq common.SendDealDataBotRequest
	botReq.DealID = req.DealID
	botReq.UserID = customerID

	// telegram/listener.go
	s.bridge.Post <- botReq

	return c.JSON(http.StatusAccepted, "ok")
}

// SendDealBrief
// @Summary      Send deal brief (stub)
// @Description  Placeholder for brief upload. Not implemented.
// @Tags         deals
// @Accept       json
// @Produce      json
// @Param        input body common.DealRequest true "Deal ID"
// @Success      200  {string}  string "ok"
// @Failure      400  {object}  string "Invalid request"
// @Failure      401  {object}  string "Unauthorized"
// @Failure      404  {object}  string "Deal not found"
// @Security     BearerAuth
// @Router       /deals/send-deal-brief [post]
func (s *Server) SendDealBrief(_ echo.Context) error {
	return nil
}

// MakeDeal
// @Summary      Make deal
// @Description  Confirm deal creation and set publication time
// @Tags         deals
// @Accept       json
// @Produce      json
// @Param        input body common.MakeDealRequest true "Deal confirmation"
// @Success      202  {string}  string "ok"
// @Failure      400  {object}  string "Invalid request"
// @Failure      404  {object}  string "Deal not found"
// @Failure      406  {object}  string "Deal data empty"
// @Failure      500  {object}  string "Internal error"
// @Security     BearerAuth
// @Router       /deals/make-deal [post]
func (s *Server) MakeDeal(c echo.Context) error {
	var req common.MakeDealRequest
	if err := c.Bind(&req); err != nil {
		s.log.Errorf("bind request body: %v", err)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		message := s.GetUserMessage(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": message})
	}

	deal := c.Get(ctxDeal).(*ent.Deal)

	if deal.Status != int(deal_status.DRAFT) {
		s.log.Errorf("make-deal: deal %s status %d is not DRAFT", deal.ID, deal.Status)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}

	if deal.Target == nil {
		s.log.Errorf("deal target is required")
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}

	if err := common.ValidatePreferableDatetime(req.PreferableDatetime); err != nil {
		s.log.Errorf("validate preferable_datetime: %v", err)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": err.Error()})
	}

	duration := common.GetDealStatusDuration(deal_status.PENDING)

	now := time.Now()
	expAt := now.Add(duration)
	status := int(deal_status.PENDING)

	if expAt.Before(time.Now()) {
		status = int(deal_status.EXPIRED)
	}

	err := s.customers.LockBalanceForDeal(c.Request().Context(), deal)
	if err != nil {
		s.log.Errorf("lock balance for deal: %v", err)
		status = common.GetHTTPStatus(common.ErrInsufficientBalance)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInsufficientBalance)})
	}

	// Merge preferable_datetime into target
	target := make(map[string]interface{})
	for k, v := range deal.Target {
		target[k] = v
	}
	if req.PreferableDatetime != nil {
		data, _ := json.Marshal(req.PreferableDatetime)
		var pdMap map[string]interface{}
		if json.Unmarshal(data, &pdMap) == nil {
			target["preferable_datetime"] = pdMap
		}
	}

	_, err = s.deals.UpdateDeal(c.Request().Context(), req.DealID, func(u *ent.DealUpdateOne) {
		u.SetTarget(target)
		u.SetExpiresAt(expAt)
		u.SetUpdatedAt(now)
		u.SetStatus(status)
	})

	if err != nil {
		s.log.Errorf("update deal: %+v", err)
		httpStatus := common.GetHTTPStatus(common.ErrInternalError)
		message := s.GetUserMessage(common.ErrInternalError)
		return c.JSON(httpStatus, map[string]string{"error": message})
	}

	// Send notification to channel manager
	locale := "en" // Default, can use deal.Edges.ChannelManager.TgLanguage if available
	if deal.Edges.ChannelManager != nil && deal.Edges.ChannelManager.TgLanguage != "" {
		locale = deal.Edges.ChannelManager.TgLanguage
	}

	dealType := deal.TargetType
	price := fmt.Sprintf("%.2f", deal.TonPrice)
	priceSpoiler := messages.WrapSpoiler(price + " TON")
	// Use Render() and escape parameters for formatted message
	text, err := s.messages.Render(locale, "notification", "new_incoming_deal", map[string]interface{}{
		"DealID":       util.EscapeMarkdownV2(deal.ID.String()),
		"DealType":     util.EscapeMarkdownV2(dealType),
		"PriceSpoiler": priceSpoiler,
	})
	if err != nil {
		s.log.Errorf("render new_incoming_deal message: %v", err)
		text = fmt.Sprintf("New incoming deal: %s", deal.ID.String())
	}

	var notify common.Notification
	notify.UserID = deal.Edges.ChannelManager.TgID
	notify.Text = text
	notify.WebAppURL = s.cfg.Bot.WebAppURL + "/deal/" + deal.ID.String()
	s.bridge.Notification <- notify

	return c.JSON(http.StatusAccepted, "ok")
}

// ListDeals lists all deals where user is advertiser or channel manager
// @Summary      List deals
// @Description  Get user's deals with pagination (as advertiser or channel manager) with channel data
// @Tags         deals
// @Accept       json
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        take query int false "Items per page (max 100)" default(20)
// @Param        order query string false "Sort order (ASC or DESC)" default(DESC)
// @Success      200  {object}  common.ListDealsWithChannelResponse
// @Failure      400  {object}  string "Invalid request"
// @Failure      401  {object}  string "Unauthorized"
// @Failure      500  {object}  string "Internal error"
// @Security     BearerAuth
// @Router       /deals/list [get]
func (s *Server) ListDeals(c echo.Context) error {
	var req common.ListDealsRequest
	if err := c.Bind(&req); err != nil {
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		message := s.GetUserMessage(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": message})
	}

	page, take, order, err := common.ValidateAndNormalizePageOptions(req.PageOptions.Page, req.PageOptions.Take, req.PageOptions.Order)
	if err != nil {
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": err.Error()})
	}
	req.PageOptions.Page = page
	req.PageOptions.Take = take
	req.PageOptions.Order = order

	customerID, err := s.GetContextUser(c)
	if err != nil {
		status := common.GetHTTPStatus(common.ErrUnauthorized)
		message := s.GetUserMessage(common.ErrUnauthorized)
		return c.JSON(status, map[string]string{"error": message})
	}

	ctx := c.Request().Context()
	result, err := s.deals.GetUserDealsPaginated(ctx, customerID, req.PageOptions)
	if err != nil {
		s.log.Errorf("get user deals: %+v", err)
		status := common.GetHTTPStatus(common.ErrInternalError)
		message := s.GetUserMessage(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": message})
	}

	response := &common.ListDealsWithChannelResponse{
		Data:  common.MapDealsWithChannel(result.Data),
		Count: result.Count,
	}

	return c.JSON(http.StatusOK, response)
}

// CancelDeal
// @Summary      Cancel deal
// @Description  Cancel deal. Status is set to CANCELED immediately. Balance is refunded asynchronously.
// @Tags         deals
// @Accept       json
// @Produce      json
// @Param        input body common.DealRequest true "Deal ID"
// @Success      202  {string}  string "ok"
// @Failure      400  {object}  string "Invalid request"
// @Failure      401  {object}  string "Unauthorized"
// @Failure      404  {object}  string "Deal not found"
// @Failure      500  {object}  string "Internal error"
// @Security     BearerAuth
// @Router       /deals/cancel-deal [post]
func (s *Server) CancelDeal(c echo.Context) error {
	var req common.DealRequest
	err := c.Bind(&req)
	if err != nil {
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}

	deal := c.Get(ctxDeal).(*ent.Deal)

	cancelAllowedStatuses := map[int]bool{
		int(deal_status.DRAFT):                  true,
		int(deal_status.PENDING):                true,
		int(deal_status.IN_PROGRESS):            true,
		int(deal_status.DISCUSSION):             true,
		int(deal_status.AWAITING_APPROVAL):      true,
		int(deal_status.AWAITING_APPROVAL_TIME): true,
		int(deal_status.AWAITING_PUBLICATION):   true,
	}
	if !cancelAllowedStatuses[deal.Status] {
		s.log.Errorf("cancel-deal: deal %s status %d not allowed for cancel", deal.ID, deal.Status)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}

	_, err = s.deals.UpdateDeal(c.Request().Context(), req.DealID, func(u *ent.DealUpdateOne) {
		u.SetStatus(int(deal_status.CANCELED))
	})

	return c.JSON(http.StatusAccepted, "ok")
}

// AcceptDeal
// @Summary      Accept deal
// @Description  Channel manager accepts the deal. Status changes to IN_PROGRESS. Parties can then message each other via deal-message.
// @Tags         deals
// @Accept       json
// @Produce      json
// @Param        input body common.DealRequest true "Deal ID"
// @Success      202  {string}  string "ok"
// @Failure      400  {object}  string "Invalid request"
// @Failure      401  {object}  string "Unauthorized"
// @Failure      404  {object}  string "Deal not found"
// @Failure      500  {object}  string "Internal error"
// @Security     BearerAuth
// @Router       /deals/accept-deal [post]
func (s *Server) AcceptDeal(c echo.Context) error {
	var req common.DealRequest
	err := c.Bind(&req)
	if err != nil {
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}

	deal := c.Get(ctxDeal).(*ent.Deal)

	if deal.Status != int(deal_status.PENDING) {
		s.log.Errorf("accept-deal: deal %s status %d is not PENDING", deal.ID, deal.Status)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}

	_, err = s.deals.UpdateDeal(c.Request().Context(), req.DealID, func(u *ent.DealUpdateOne) {
		u.SetStatus(int(deal_status.IN_PROGRESS))
	})
	if err != nil {
		s.log.Errorf("update deal status: %v", err)
		status := common.GetHTTPStatus(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInternalError)})
	}

	// Send notification to advertiser that deal was accepted
	if deal.Edges.Advertiser != nil {
		locale := "en"
		if deal.Edges.Advertiser.TgLanguage != "" {
			locale = deal.Edges.Advertiser.TgLanguage
		}

		channelName := ""
		if deal.Edges.Channel != nil {
			channelName = deal.Edges.Channel.TgName
			if channelName == "" {
				channelName = deal.Edges.Channel.TgUsername
			}
		}

		// Use Render() and escape parameters for formatted message
		text, err := s.messages.Render(locale, "notification", "deal_accepted", map[string]interface{}{
			"DealID":      util.EscapeMarkdownV2(deal.ID.String()),
			"ChannelName": util.EscapeMarkdownV2(channelName),
		})
		if err != nil {
			s.log.Errorf("render deal_accepted message: %v", err)
			text = fmt.Sprintf("Deal %s has been accepted!", deal.ID.String())
		}

		var notify common.Notification
		notify.UserID = deal.Edges.Advertiser.TgID
		notify.Text = text
		notify.WebAppURL = s.cfg.Bot.WebAppURL + "/deal/" + deal.ID.String()
		s.bridge.Notification <- notify
	}

	return c.JSON(http.StatusAccepted, "ok")
}

// GetDeal gets a deal by ID
// @Summary      Get deal
// @Description  Get deal details by ID with channel data
// @Tags         deals
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Deal ID"
// @Success      200  {object}  common.DealWithChannelResponse
// @Failure      400  {object}  string "Invalid ID"
// @Failure      404  {object}  string "Deal not found"
// @Failure      500  {object}  string "Internal error"
// @Security     BearerAuth
// @Router       /deals/{id} [get]
func (s *Server) GetDeal(c echo.Context) error {
	deal := c.Get(ctxDeal).(*ent.Deal)

	return c.JSON(http.StatusOK, common.MapDealWithChannel(deal))
}

// GetDealData
// @Summary      Get deal data
// @Description  Request deal info and post content to be sent to the user in Telegram. Returns deal target (post data) in response.
// @Tags         deals
// @Accept       json
// @Produce      json
// @Param        input body common.DealRequest true "Deal ID"
// @Success      200  {object}  common.DealTarget "Deal target (post data)"
// @Failure      400  {object}  string "Invalid request"
// @Failure      401  {object}  string "Unauthorized"
// @Failure      404  {object}  string "Deal not found"
// @Failure      500  {object}  string "Internal error"
// @Security     BearerAuth
// @Router       /deals/get-deal-data [post]
func (s *Server) GetDealData(c echo.Context) error {
	deal := c.Get(ctxDeal).(*ent.Deal)

	userID, err := s.GetContextUser(c)
	if err != nil {
		s.log.Errorf("get-deal-data: get context user: %v", err)
		status := common.GetHTTPStatus(common.ErrUnauthorized)
		message := s.GetUserMessage(common.ErrUnauthorized)
		return c.JSON(status, map[string]string{"error": message})
	}

	mapped := common.MapDeal(deal)
	if mapped == nil {
		s.log.Errorf("get-deal-data: failed to map deal %s", deal.ID)
		status := common.GetHTTPStatus(common.ErrInternalError)
		message := s.GetUserMessage(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": message})
	}

	var botReq common.ShowDealDataBotRequest
	botReq.DealID = deal.ID
	botReq.UserID = userID

	s.bridge.Deal <- botReq

	return c.JSON(http.StatusOK, mapped.Target)
}

// DealMessage
// @Summary      Send deal message
// @Description  Send a message between deal parties
// @Tags         deals
// @Accept       json
// @Produce      json
// @Param        input body common.DealMessageRequest true "Message"
// @Success      201  {string}  string "ok"
// @Failure      400  {object}  string "Invalid request"
// @Failure      404  {object}  string "Deal not found"
// @Failure      500  {object}  string "Internal error"
// @Security     BearerAuth
// @Router       /deals/deal-message [post]
func (s *Server) DealMessage(c echo.Context) error {
	var req common.DealMessageRequest
	if err := c.Bind(&req); err != nil {
		s.log.Errorf("bind body: %v", err)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		message := s.GetUserMessage(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": message})
	}

	customerID, err := s.GetContextUser(c)
	if err != nil {
		s.log.Errorf("get ctx user: %v", err)
		status := common.GetHTTPStatus(common.ErrUnauthorized)
		message := s.GetUserMessage(common.ErrUnauthorized)
		return c.JSON(status, map[string]string{"error": message})
	}

	deal := c.Get(ctxDeal).(*ent.Deal)

	allowedStatuses := map[int]bool{
		int(deal_status.IN_PROGRESS):            true,
		int(deal_status.DISCUSSION):             true,
		int(deal_status.AWAITING_APPROVAL):      true,
		int(deal_status.AWAITING_APPROVAL_TIME): true,
		int(deal_status.AWAITING_PUBLICATION):   true,
		int(deal_status.PUBLISHED):              true,
	}
	if !allowedStatuses[deal.Status] {
		s.log.Errorf("deal status: %d not allowed for deal-message", deal.Status)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}

	var botReq common.DealMessageBotRequest
	botReq.DealID = req.DealID

	switch customerID {
	case deal.AdvertiserCustomerID:
		botReq.SenderID = customerID
		botReq.RecipientID = deal.ChannelManagerID
	case deal.ChannelManagerID:
		botReq.SenderID = customerID
		botReq.RecipientID = deal.AdvertiserCustomerID
	default:
		s.log.Errorf("deal id: %s, not in customer", deal.ID)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}

	s.bridge.Dialog <- botReq

	return c.JSON(http.StatusCreated, "ok")
}

// ApproveDealData
// @Summary      Approve deal data
// @Description  Approve deal and move to AWAITING_PUBLICATION
// @Tags         deals
// @Accept       json
// @Produce      json
// @Param        input body common.DealRequest true "Deal ID"
// @Success      202  {string}  string "ok"
// @Failure      400  {object}  string "Invalid request"
// @Failure      401  {object}  string "Unauthorized"
// @Failure      404  {object}  string "Deal not found"
// @Failure      500  {object}  string "Internal error"
// @Security     BearerAuth
// @Router       /deals/approve-deal-data [post]
func (s *Server) ApproveDealData(c echo.Context) error {
	var req common.DealRequest
	err := c.Bind(&req)
	if err != nil {
		s.log.Errorf("bind body: %v", err)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		message := s.GetUserMessage(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": message})
	}

	deal := c.Get(ctxDeal).(*ent.Deal)

	if deal.Status != int(deal_status.AWAITING_APPROVAL_TIME) {
		s.log.Errorf("approve-deal-data: deal %s status %d is not AWAITING_APPROVAL_TIME", deal.ID, deal.Status)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}

	if deal.PublicationTime == nil {
		s.log.Errorf("approve-deal-data: deal %s has nil publication_time", deal.ID)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}

	_, topHours, commonHours, err := deal_target_type.ParseDealTargetTypeToParams(deal.TargetType)
	if err != nil {
		s.log.Errorf("approve-deal-data: invalid target_type %q for deal %s: %v", deal.TargetType, deal.ID, err)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}

	pub := deal.PublicationTime.UTC()
	topDeadline := pub.Add(time.Duration(topHours) * time.Hour)
	commonDeadline := pub.Add(time.Duration(commonHours) * time.Hour)

	_, err = s.deals.UpdateDeal(c.Request().Context(), req.DealID, func(u *ent.DealUpdateOne) {
		u.SetStatus(int(deal_status.AWAITING_PUBLICATION))
		u.SetTopDeadline(topDeadline)
		u.SetCommonDeadline(commonDeadline)
	})

	if err != nil {
		s.log.Errorf("update deal: %+v", err)
		status := common.GetHTTPStatus(common.ErrInternalError)
		message := s.GetUserMessage(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": message})
	}

	deal, err = s.deals.GetDeal(c.Request().Context(), req.DealID)
	if err == nil && deal.PublicationTime != nil && deal.Edges.Advertiser != nil && deal.Edges.ChannelManager != nil {
		pubTimeStr := deal.PublicationTime.Format("02 Jan 2006, 15:04 UTC")
		channelName := "—"
		if deal.Edges.Channel != nil && deal.Edges.Channel.TgUsername != "" {
			channelName = "@" + deal.Edges.Channel.TgUsername
		} else if deal.Edges.Channel != nil {
			channelName = deal.Edges.Channel.TgName
		}
		params := map[string]interface{}{
			"DealID":          util.EscapeMarkdownV2(deal.ID.String()),
			"ChannelName":     util.EscapeMarkdownV2(channelName),
			"PublicationTime": util.EscapeMarkdownV2(pubTimeStr),
		}
		webAppURL := s.cfg.Bot.WebAppURL + "/deal/" + deal.ID.String()

		for _, recipient := range []struct {
			tgID   int64
			locale string
		}{
			{deal.Edges.Advertiser.TgID, deal.Edges.Advertiser.TgLanguage},
			{deal.Edges.ChannelManager.TgID, deal.Edges.ChannelManager.TgLanguage},
		} {
			locale := recipient.locale
			if locale == "" {
				locale = "en"
			}
			text, err := s.messages.Render(locale, "notification", "post_scheduled", params)
			if err != nil {
				s.log.Errorf("render post_scheduled message: %v", err)
				text = fmt.Sprintf("Post scheduled for deal %s on %s", deal.ID.String(), pubTimeStr)
			}
			notify := common.Notification{
				UserID:    recipient.tgID,
				Text:      text,
				WebAppURL: webAppURL,
			}
			s.bridge.Notification <- notify
		}
	}

	return c.JSON(http.StatusAccepted, "ok")
}

func (s *Server) ReviseDealData(c echo.Context) error {
	var req common.DealRequest
	err := c.Bind(&req)
	if err != nil {
		s.log.Errorf("revise-deal-data: invalid request body: %v", err)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		message := s.GetUserMessage(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": message})
	}

	customerID, err := s.GetContextUser(c)
	if err != nil {
		s.log.Errorf("revise-deal-data: invalid user id: %v", err)
		status := common.GetHTTPStatus(common.ErrUnauthorized)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrUnauthorized)})
	}

	deal, err := s.deals.GetDeal(c.Request().Context(), req.DealID)
	if err != nil {
		s.log.Errorf("get deal: %v", err)
		status := common.GetHTTPStatus(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInternalError)})
	}

	_, err = s.deals.UpdateDeal(c.Request().Context(), req.DealID, func(u *ent.DealUpdateOne) {
		u.SetStatus(int(deal_status.DISCUSSION))
	})

	if err != nil {
		s.log.Errorf("update deal: %+v", err)
		status := common.GetHTTPStatus(common.ErrInternalError)
		message := s.GetUserMessage(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": message})
	}

	// Notify the other party about revision request
	var recipientID int64
	var locale string

	// Determine who should receive notification (the other party)
	if deal.AdvertiserCustomerID == customerID {
		// Advertiser initiated revision, notify manager
		if deal.Edges.ChannelManager != nil {
			recipientID = deal.Edges.ChannelManager.TgID
			locale = deal.Edges.ChannelManager.TgLanguage
		}
	} else {
		// Manager initiated revision, notify advertiser
		if deal.Edges.Advertiser != nil {
			recipientID = deal.Edges.Advertiser.TgID
			locale = deal.Edges.Advertiser.TgLanguage
		}
	}

	if recipientID > 0 {
		if locale == "" {
			locale = "en"
		}

		var text string
		text, err = s.messages.Render(locale, "notification", "deal_revision_requested", map[string]interface{}{
			"DealID": util.EscapeMarkdownV2(deal.ID.String()),
		})
		if err != nil {
			s.log.Errorf("render deal_revision_requested message: %v", err)
			text = fmt.Sprintf("Deal %s: revision requested", deal.ID.String())
		}

		var notify common.Notification
		notify.UserID = recipientID
		notify.Text = text
		notify.WebAppURL = s.cfg.Bot.WebAppURL + "/deal/" + deal.ID.String()
		s.bridge.Notification <- notify
	}

	return c.JSON(http.StatusAccepted, "ok")
}

// DeclineDeal
// @Summary      Decline deal
// @Description  Decline deal. Channel manager or advertiser declines the deal. Status changes.
// @Tags         deals
// @Accept       json
// @Produce      json
// @Param        input body common.DealRequest true "Deal ID"
// @Success      202  {string}  string "ok"
// @Failure      400  {object}  string "Invalid request"
// @Failure      401  {object}  string "Unauthorized"
// @Failure      404  {object}  string "Deal not found"
// @Failure      500  {object}  string "Internal error"
// @Security     BearerAuth
// @Router       /deals/decline-deal [post]
func (s *Server) DeclineDeal(c echo.Context) error {
	var req common.DealRequest
	err := c.Bind(&req)
	if err != nil {
		s.log.Errorf("decline-deal: invalid request body: %v", err)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}

	deal := c.Get(ctxDeal).(*ent.Deal)

	if deal.Status != int(deal_status.AWAITING_APPROVAL_TIME) {
		s.log.Errorf("decline-deal: deal %s status %d is not AWAITING_APPROVAL_TIME", deal.ID, deal.Status)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}

	_, err = s.deals.UpdateDeal(c.Request().Context(), req.DealID, func(u *ent.DealUpdateOne) {
		u.SetStatus(int(deal_status.IN_PROGRESS))
	})
	if err != nil {
		s.log.Errorf("decline-deal update: %v", err)
		status := common.GetHTTPStatus(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInternalError)})
	}

	// Notify channel manager (who proposed the time) that it was declined
	deal, err = s.deals.GetDeal(c.Request().Context(), req.DealID)
	if err == nil && deal.Edges.ChannelManager != nil {
		locale := "en"
		if deal.Edges.ChannelManager.TgLanguage != "" {
			locale = deal.Edges.ChannelManager.TgLanguage
		}
		channelName := "—"
		if deal.Edges.Channel != nil && deal.Edges.Channel.TgUsername != "" {
			channelName = "@" + deal.Edges.Channel.TgUsername
		}
		var text string
		text, err = s.messages.Render(locale, "notification", "time_declined", map[string]interface{}{
			"DealID":      util.EscapeMarkdownV2(deal.ID.String()),
			"ChannelName": util.EscapeMarkdownV2(channelName),
		})
		if err != nil {
			s.log.Errorf("render time_declined message: %v", err)
			text = fmt.Sprintf("Proposed time declined for deal %s. Please discuss in chat and propose a new time.", deal.ID.String())
		}

		var notify common.Notification
		notify.UserID = deal.Edges.ChannelManager.TgID
		notify.Text = text
		notify.WebAppURL = s.cfg.Bot.WebAppURL + "/deal/" + deal.ID.String()

		s.bridge.Notification <- notify
	}

	return c.JSON(http.StatusAccepted, "ok")
}

// ApproveAndProposeTime
// @Summary      Approve and propose publication time
// @Description  Channel manager approves deal data and proposes publication time (RFC3339). Status changes to AWAITING_APPROVAL_TIME. Does not modify target or preferable_datetime.
// @Tags         deals
// @Accept       json
// @Produce      json
// @Param        input body common.ApproveAndProposeTimeRequest true "Deal ID and publication time (RFC3339)"
// @Success      202  {string}  string "ok"
// @Failure      400  {object}  string "Invalid request"
// @Failure      401  {object}  string "Unauthorized"
// @Failure      404  {object}  string "Deal not found"
// @Failure      500  {object}  string "Internal error"
// @Security     BearerAuth
// @Router       /deals/approve-and-propose-time [post]
func (s *Server) ApproveAndProposeTime(c echo.Context) error {
	var req common.ApproveAndProposeTimeRequest
	if err := c.Bind(&req); err != nil {
		s.log.Errorf("bind request body: %v", err)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		message := s.GetUserMessage(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": message})
	}

	deal := c.Get(ctxDeal).(*ent.Deal)

	if deal.Status != int(deal_status.IN_PROGRESS) {
		s.log.Errorf("approve-and-propose-time: deal %s status %d is not IN_PROGRESS", req.DealID, deal.Status)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}

	publicationTime, err := common.ValidatePublicationTime(req.PublicationTime)
	if err != nil {
		s.log.Errorf("validate publication_time: %v", err)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": err.Error()})
	}

	duration := common.GetDealStatusDuration(deal_status.AWAITING_APPROVAL_TIME)
	now := time.Now()
	expAt := now.Add(duration)
	status := int(deal_status.AWAITING_APPROVAL_TIME)
	if expAt.Before(now) {
		status = int(deal_status.EXPIRED)
	}

	_, err = s.deals.UpdateDeal(c.Request().Context(), req.DealID, func(u *ent.DealUpdateOne) {
		u.SetPublicationTime(publicationTime)
		u.SetExpiresAt(expAt)
		u.SetUpdatedAt(now)
		u.SetStatus(status)
	})
	if err != nil {
		s.log.Errorf("update deal: %+v", err)
		httpStatus := common.GetHTTPStatus(common.ErrInternalError)
		message := s.GetUserMessage(common.ErrInternalError)
		return c.JSON(httpStatus, map[string]string{"error": message})
	}

	// Reload deal with edges to send notification to advertiser
	deal, err = s.deals.GetDeal(c.Request().Context(), req.DealID)
	if err == nil && deal.Edges.Advertiser != nil {
		locale := "en"
		if deal.Edges.Advertiser.TgLanguage != "" {
			locale = deal.Edges.Advertiser.TgLanguage
		}
		channelName := "—"
		if deal.Edges.Channel != nil && deal.Edges.Channel.TgUsername != "" {
			channelName = "@" + deal.Edges.Channel.TgUsername
		}
		pubTimeStr := publicationTime.Format("02 Jan 2006, 15:04 UTC")
		text, err := s.messages.Render(locale, "notification", "time_proposed", map[string]interface{}{
			"DealID":          util.EscapeMarkdownV2(deal.ID.String()),
			"ChannelName":     util.EscapeMarkdownV2(channelName),
			"PublicationTime": util.EscapeMarkdownV2(pubTimeStr),
		})
		if err != nil {
			s.log.Errorf("render time_proposed message: %v", err)
			text = fmt.Sprintf("Publication time proposed for deal %s: %s", deal.ID.String(), pubTimeStr)
		}
		var notify common.Notification
		notify.UserID = deal.Edges.Advertiser.TgID
		notify.Text = text
		notify.WebAppURL = s.cfg.Bot.WebAppURL + "/deal/" + deal.ID.String()
		s.bridge.Notification <- notify
	}

	return c.JSON(http.StatusAccepted, "ok")
}
