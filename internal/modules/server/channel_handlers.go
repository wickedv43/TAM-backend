package server

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/wickedv43/TAM-backend/internal/common"
	"github.com/wickedv43/TAM-backend/internal/common/constants/channel_status"
	"github.com/wickedv43/TAM-backend/internal/common/constants/customer_channel_role"
	"github.com/wickedv43/TAM-backend/internal/database/ent"
)

// AddChannel creates a new channel
// @Summary      Add channel
// @Description  Add a new Telegram channel
// @Tags         channels
// @Accept       json
// @Produce      json
// @Param        input body common.AddChannelRequest true "Channel details"
// @Success      201  {object}  common.CreateChannelResponse
// @Failure      400  {object}  string "Invalid request"
// @Failure      401  {object}  string "Unauthorized"
// @Failure      403  {object}  string "Forbidden"
// @Failure      409  {object}  string "Conflict"
// @Failure      500  {object}  string "Internal error"
// @Security     BearerAuth
// @Router       /channels/add-channel [post]
func (s *Server) AddChannel(c echo.Context) error {
	customerID, err := s.GetContextUser(c)
	if err != nil {
		s.log.Errorf("getting customer id: %s", err)
		status := common.GetHTTPStatus(common.ErrUnauthorized)
		message := s.GetUserMessage(common.ErrUnauthorized)
		return c.JSON(status, map[string]string{"error": message})
	}

	ctx := c.Request().Context()

	tgUser, err := s.customers.GetCustomer(ctx, customerID)
	if err != nil {
		s.log.Errorf("getting customer: %s", err)
		status := common.GetHTTPStatus(common.ErrUnauthorized)
		message := s.GetUserMessage(common.ErrUnauthorized)
		return c.JSON(status, map[string]string{"error": message})
	}

	var req common.AddChannelRequest
	if err = c.Bind(&req); err != nil {
		s.log.Errorf("binding request: %s", err)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		message := s.GetUserMessage(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": message})
	}

	var statsReq common.StatsBotAddChannelRequest
	statsReq.ChannelUsername = parseChannelUsername(req.ChannelUsername)
	statsReq.UserTgID = tgUser.TgID

	statsResp, err := s.statsBot.GetChannel(statsReq)
	if err != nil {
		s.log.Errorf("failed to get channel from stats bot: %+v", err)
		status := common.GetHTTPStatus(err)
		message := s.GetUserMessage(err)

		return c.JSON(status, map[string]string{
			"error": message,
		})
	}

	existingChannel, err := s.channels.GetChannelByTgID(ctx, statsResp.TgID)
	if err != nil && !ent.IsNotFound(err) {
		s.log.Errorf("check channel existence: %+v", err)
		status := common.GetHTTPStatus(common.ErrInternalError)
		message := s.GetUserMessage(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": message})
	}

	if existingChannel != nil {
		var (
			updatedChannel  *ent.Channel
			customerChannel *ent.CustomerChannel
		)
		customerChannel, err = s.customers.GetCustomerChannel(ctx, customerID, existingChannel.ID)
		if err != nil {
			if ent.IsNotFound(err) {
				s.log.Errorf("customer channel not found: %s", existingChannel.ID)
				status := common.GetHTTPStatus(common.ErrChannelExists)
				return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrChannelExists)})
			}
			s.log.Errorf("get customer channel: %+v", err)
			status := common.GetHTTPStatus(common.ErrInternalError)
			message := s.GetUserMessage(common.ErrInternalError)
			return c.JSON(status, map[string]string{"error": message})
		}

		if existingChannel.Status == int(channel_status.UNACTIVATED) {
			s.log.Infof("Updating unactivated channel %d for user %s", existingChannel.ID, customerID)
			newStatus := int(channel_status.UNACTIVATED)
			if statsResp.StatsAvailable {
				newStatus = int(channel_status.AWAITING_DATA)
			}

			userRole := int(customer_channel_role.ADMIN)
			if statsResp.UserRole == "creator" {
				userRole = int(customer_channel_role.OWNER)
			}

			updatedChannel, err = s.channels.UpdateChannel(ctx, existingChannel.ID, func(u *ent.ChannelUpdateOne) {
				u.SetTgUsername(statsResp.Username)
				u.SetTgName(statsResp.Title)
				u.SetTgDescription(statsResp.About)
				u.SetTgPicture(statsResp.PhotoHash)
				u.SetSubscribers(statsResp.Subscribers)
				u.SetPremiumSubscribers(statsResp.PremiumSubscribers)
				u.SetMedianPostViews(statsResp.MedianViews)
				u.SetAvgPostViews(statsResp.AverageViews)
				u.SetStatus(newStatus)
				u.SetStatsUpdatedAt(time.Now())
				u.SetNotificationsOn(statsResp.NotificationsOn)
				u.SetFirstPostDate(statsResp.FirstPostDate)
				u.SetTotalPosts(statsResp.TotalPosts)

				if statsResp.StatsAvailable && statsResp.Stats != nil {
					u.SetStats(common.ChannelStatsToMap(statsResp.Stats))
				}
			})

			if err != nil {
				s.log.Errorf("update channel: %+v", err)
				status := common.GetHTTPStatus(common.ErrInternalError)
				message := s.GetUserMessage(common.ErrInternalError)
				return c.JSON(status, map[string]string{"error": message})
			}

			if customerChannel.Role != userRole {
				_, err = s.customers.UpdateCustomerChannelRole(ctx, customerID, existingChannel.ID, userRole, nil)
				if err != nil {
					s.log.Warnf("failed to update customer channel role: %+v", err)
				}
			}

			return c.JSON(http.StatusOK, common.MapChannelWithoutStats(updatedChannel))
		}

		status := common.GetHTTPStatus(common.ErrChannelExists)
		message := s.GetUserMessage(common.ErrChannelExists)
		return c.JSON(status, map[string]string{"error": message})
	}

	channelStatus := int(channel_status.UNACTIVATED)
	if statsResp.StatsAvailable {
		channelStatus = int(channel_status.AWAITING_DATA)
	}

	channel := &ent.Channel{
		TgID:               -(1000000000000 + statsResp.TgID),
		TgUsername:         statsResp.Username,
		TgName:             statsResp.Title,
		TgDescription:      statsResp.About,
		TgPicture:          statsResp.PhotoHash,
		Status:             channelStatus,
		IsListed:           false,
		Subscribers:        statsResp.Subscribers,
		PremiumSubscribers: statsResp.PremiumSubscribers,
		MedianPostViews:    statsResp.MedianViews,
		AvgPostViews:       statsResp.AverageViews,
		NotificationsOn:    statsResp.NotificationsOn,
		FirstPostDate:      statsResp.FirstPostDate,
		TotalPosts:         statsResp.TotalPosts,
	}

	if statsResp.StatsAvailable && statsResp.Stats != nil {
		channel.Stats = common.ChannelStatsToMap(statsResp.Stats)
	}

	createdChannel, err := s.channels.CreateChannel(ctx, channel)
	if err != nil {
		s.log.Errorf("create channel: %+v", err)
		status := common.GetHTTPStatus(common.ErrInternalError)
		message := s.GetUserMessage(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": message})
	}

	userRole := int(customer_channel_role.ADMIN)
	if statsResp.UserRole == "creator" {
		userRole = int(customer_channel_role.OWNER)
	}

	_, err = s.customers.AddCustomerChannel(ctx, customerID, createdChannel.ID, userRole, nil)
	if err != nil {
		s.log.Errorf("add customer channel: %+v", err)
		_ = s.channels.DeleteChannel(ctx, createdChannel.ID)
		status := common.GetHTTPStatus(common.ErrInternalError)
		message := s.GetUserMessage(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": message})
	}

	return c.JSON(http.StatusCreated, common.MapChannelWithoutStats(createdChannel))
}

// ProvideChannelData
// @Summary      Provide channel data
// @Description  Update channel details (description, tags, etc.)
// @Tags         channels
// @Accept       json
// @Produce      json
// @Param        input body common.ProvideChannelDataRequest true "Channel data"
// @Success      200  {object}  common.ChannelResponse
// @Failure      400  {object}  string "Invalid request"
// @Failure      401  {object}  string "Unauthorized"
// @Failure      403  {object}  string "Forbidden"
// @Failure      404  {object}  string "Channel not found"
// @Failure      500  {object}  string "Internal error"
// @Security     BearerAuth
// @Router       /channels/provide-channel-data [post]
func (s *Server) ProvideChannelData(c echo.Context) error {
	var req common.ProvideChannelDataRequest
	if err := c.Bind(&req); err != nil {
		s.log.Errorf("bind request: %+v", err)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		message := s.GetUserMessage(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": message})
	}

	ctx := c.Request().Context()

	channel, err := s.channels.GetChannel(ctx, req.ChannelID)
	if err != nil {
		if ent.IsNotFound(err) {
			s.log.Errorf("channel '%s' not found", req.ChannelID)
			status := common.GetHTTPStatus(common.ErrChannelNotFound)
			message := s.GetUserMessage(common.ErrChannelNotFound)
			return c.JSON(status, map[string]string{"error": message})
		}
		s.log.Errorf("get channel: %+v", err)
		status := common.GetHTTPStatus(common.ErrInternalError)
		message := s.GetUserMessage(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": message})
	}

	if channel.Status != int(channel_status.AWAITING_DATA) {
		s.log.Errorf("channel '%s' is not AWAITING_DATA", req.ChannelID)
		status := common.GetHTTPStatus(common.ErrChannelStatusInvalid)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrChannelStatusInvalid)})
	}

	descLen := len(req.Description)
	if descLen < common.MIN_DESCRIPTION_LENGTH {
		s.log.Errorf("channel desc '%s' is too short", req.ChannelID)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}
	if descLen > common.MAX_DESCRIPTION_LENGTH {
		s.log.Errorf("channel desc '%s' is too long", req.ChannelID)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}

	if len(req.Tags) < common.MIN_TAGS_COUNT {
		s.log.Errorf("channel min tags '%s' is too short", req.ChannelID)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}
	if len(req.Tags) > common.MAX_TAGS_COUNT {
		s.log.Errorf("channel max tags '%s' is too long", req.ChannelID)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}
	for _, tag := range req.Tags {
		if !common.IsValidTag(tag) {
			s.log.Errorf("channel tag '%s' is invalid", req.ChannelID)
			status := common.GetHTTPStatus(common.ErrInvalidRequest)
			return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
		}
	}

	if !common.IsValidLanguage(req.Language) {
		s.log.Errorf("channel language '%s' is invalid", req.Language)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}

	if err = common.ValidatePrice("Post_1_24", req.Prices.Post_1_24, true); err != nil {
		s.log.Errorf("channel price invalid: %+v", err)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}
	if err = common.ValidatePrice("Post_2_48", req.Prices.Post_2_48, false); err != nil {
		s.log.Errorf("channel price invalid: %+v", err)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}
	if err = common.ValidatePrice("Post_3_72", req.Prices.Post_3_72, false); err != nil {
		s.log.Errorf("channel price invalid: %+v", err)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}

	updatedChannel, err := s.channels.UpdateChannel(ctx, req.ChannelID, func(u *ent.ChannelUpdateOne) {
		if req.Description != "" {
			u.SetCommentary(req.Description)
		}
		u.SetMainLanguage(req.Language)
		u.SetTags(req.Tags)
		u.SetPrices((&req.Prices).ToMap())
		u.SetIsListed(true)
		u.SetStatus(int(channel_status.ACTIVATED))
	})

	if err != nil {
		s.log.Errorf("update channel: %+v", err)
		status := common.GetHTTPStatus(common.ErrInternalError)
		message := s.GetUserMessage(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": message})
	}

	return c.JSON(http.StatusOK, common.MapChannel(updatedChannel))
}

// ListChannels lists all channels with optional filters
// @Summary      List channels
// @Description  Get listed channels with pagination and optional filters
// @Tags         channels
// @Accept       json
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        take query int false "Items per page (max 100)" default(20)
// @Param        order query string false "Sort order (ASC or DESC)" default(DESC)
// @Param        myChannels query bool false "Filter only user's channels (admin/owner)"
// @Success      200  {object}  common.ListChannelsResponse
// @Failure      400  {object}  string "Invalid request"
// @Failure      401  {object}  string "Unauthorized"
// @Failure      500  {object}  string "Internal error"
// @Security     BearerAuth
// @Router       /channels/list-channels [get]
func (s *Server) ListChannels(c echo.Context) error {
	var req common.ListChannelsRequest
	if err := c.Bind(&req); err != nil {
		s.log.Errorf("bind request: %+v", err)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		message := s.GetUserMessage(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": message})
	}

	page, take, order, err := common.ValidateAndNormalizePageOptions(req.PageOptions.Page, req.PageOptions.Take, req.PageOptions.Order)
	if err != nil {
		s.log.Errorf("validate page options: %+v", err)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": err.Error()})
	}
	req.PageOptions.Page = page
	req.PageOptions.Take = take
	req.PageOptions.Order = order

	ctx := c.Request().Context()
	var result *common.Pagination[[]*ent.Channel]

	if req.MyChannels {
		var customerID uuid.UUID
		customerID, err = s.GetContextUser(c)
		if err != nil {
			s.log.Errorf("get customer id: %+v", err)
			status := common.GetHTTPStatus(common.ErrUnauthorized)
			message := s.GetUserMessage(common.ErrUnauthorized)
			return c.JSON(status, map[string]string{"error": message})
		}

		result, err = s.channels.GetCustomerChannelsPaginated(ctx, customerID, req.PageOptions)
		if err != nil {
			s.log.Errorf("get customer channels: %+v", err)
			status := common.GetHTTPStatus(common.ErrInternalError)
			message := s.GetUserMessage(common.ErrInternalError)
			return c.JSON(status, map[string]string{"error": message})
		}
	} else {
		result, err = s.channels.GetListedChannelsPaginated(ctx, req.PageOptions)
		if err != nil {
			s.log.Errorf("list channels: %+v", err)
			status := common.GetHTTPStatus(common.ErrInternalError)
			message := s.GetUserMessage(common.ErrInternalError)
			return c.JSON(status, map[string]string{"error": message})
		}
	}

	// @TODO: To generic. Current types parser can parse generics
	// response := common.NewPagination(
	// 	common.MapChannelsWithoutStats(result.Data),
	// 	result.Count,
	// )
	response := &common.ListChannelsResponse{
		Data:  common.MapChannelsWithoutStats(result.Data),
		Count: result.Count,
	}

	return c.JSON(http.StatusOK, response)
}

// GetChannel gets a channel by ID
// @Summary      Get channel
// @Description  Get channel details
// @Tags         channels
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Channel ID"
// @Success      200  {object}  common.GetChannelResponse
// @Failure      400  {object}  string "Invalid request"
// @Failure      404  {object}  string "Channel not found"
// @Failure      500  {object}  string "Internal error"
// @Security     BearerAuth
// @Router       /channels/{id} [get]
func (s *Server) GetChannel(c echo.Context) error {
	customerID, err := s.GetContextUser(c)
	if err != nil {
		s.log.Errorf("get customer id: %+v", err)
		status := common.GetHTTPStatus(common.ErrUnauthorized)
		message := s.GetUserMessage(common.ErrUnauthorized)
		return c.JSON(status, map[string]string{"error": message})
	}

	idStr := c.Param("id")
	channelID, err := uuid.Parse(idStr)
	if err != nil {
		s.log.Errorf("invalid channel id: %s", idStr)
		status := common.GetHTTPStatus(common.ErrInvalidUUID)
		message := s.GetUserMessage(common.ErrInvalidUUID)
		return c.JSON(status, map[string]string{"error": message})
	}

	ctx := c.Request().Context()

	channel, err := s.channels.GetChannelWithCustomerChannels(ctx, channelID)
	if err != nil {
		if ent.IsNotFound(err) {
			s.log.Errorf("channel not found: %s", channelID)
			status := common.GetHTTPStatus(common.ErrChannelNotFound)
			message := s.GetUserMessage(common.ErrChannelNotFound)
			return c.JSON(status, map[string]string{"error": message})
		}
		s.log.Errorf("get channel: %+v", err)
		status := common.GetHTTPStatus(common.ErrInternalError)
		message := s.GetUserMessage(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": message})
	}

	canEdit := false
	if channel.Edges.CustomerChannels != nil {
		for _, cc := range channel.Edges.CustomerChannels {
			if cc.CustomerID == customerID &&
				(cc.Role == int(customer_channel_role.ADMIN) ||
					cc.Role == int(customer_channel_role.OWNER)) {
				canEdit = true
				break
			}
		}
	}

	response := &common.GetChannelResponse{
		CanEdit: canEdit,
		Channel: *common.MapChannel(channel),
	}

	return c.JSON(http.StatusOK, response)
}

// DeleteChannel deletes a channel
// @Summary      Delete channel
// @Description  Delete channel by ID
// @Tags         channels
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Channel ID"
// @Success      204  {object}  nil
// @Failure      400  {object}  string "Invalid ID"
// @Failure      500  {object}  string "Internal error"
// @Security     BearerAuth
// @Router       /channels/{id} [delete]
func (s *Server) DeleteChannel(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		s.log.Errorf("invalid channel id: %s", idStr)
		status := common.GetHTTPStatus(common.ErrInvalidUUID)
		message := s.GetUserMessage(common.ErrInvalidUUID)
		return c.JSON(status, map[string]string{"error": message})
	}

	if err = s.channels.DeleteChannel(c.Request().Context(), id); err != nil {
		s.log.Errorf("delete channel: %+v", err)
		status := common.GetHTTPStatus(common.ErrInternalError)
		message := s.GetUserMessage(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": message})
	}

	return c.NoContent(http.StatusNoContent)
}
