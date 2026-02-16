package server

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/wickedv43/TAM-backend/internal/common"
	"github.com/wickedv43/TAM-backend/internal/database/ent"
)

// AddBrief creates a new brief
// @Summary      Create brief
// @Description  Creates a new brief for the user
// @Tags         briefs
// @Accept       json
// @Produce      json
// @Success      201  {object}  common.BriefResponse
// @Failure      401  {object}  string "Unauthorized"
// @Failure      500  {object}  string "Internal error"
// @Security     BearerAuth
// @Router       /briefs/add-brief [post]
func (s *Server) AddBrief(c echo.Context) error {
	userID, err := s.GetContextUser(c)
	if err != nil {
		status := common.GetHTTPStatus(common.ErrUnauthorized)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrUnauthorized)})
	}

	brief := &ent.Brief{}
	// Set fields if any

	createdBrief, err := s.briefs.CreateBrief(c.Request().Context(), brief)
	if err != nil {
		s.log.Errorf("create brief: %v", err)
		status := common.GetHTTPStatus(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInternalError)})
	}

	// Link brief to user
	_, err = s.briefs.AddBriefCustomer(c.Request().Context(), createdBrief.ID, userID)
	if err != nil {
		s.log.Errorf("link brief to user: %v", err)
		status := common.GetHTTPStatus(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInternalError)})
	}

	return c.JSON(http.StatusCreated, common.MapBrief(createdBrief))
}

// GetBrief gets a brief by ID
// @Summary      Get brief
// @Description  Get brief details by ID
// @Tags         briefs
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Brief ID"
// @Success      200  {object}  common.BriefResponse
// @Failure      400  {object}  string "Invalid ID"
// @Failure      404  {object}  string "Brief not found"
// @Failure      500  {object}  string "Internal error"
// @Security     BearerAuth
// @Router       /briefs/{id} [get]
func (s *Server) GetBrief(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		status := common.GetHTTPStatus(common.ErrInvalidUUID)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidUUID)})
	}

	brief, err := s.briefs.GetBrief(c.Request().Context(), id)
	if err != nil {
		if ent.IsNotFound(err) {
			status := common.GetHTTPStatus(common.ErrNotFound)
			return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrNotFound)})
		}
		s.log.Errorf("get brief: %v", err)
		status := common.GetHTTPStatus(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInternalError)})
	}

	return c.JSON(http.StatusOK, common.MapBrief(brief))
}

// ListBriefs lists user's briefs
// @Summary      List briefs
// @Description  List all briefs for current user
// @Tags         briefs
// @Accept       json
// @Produce      json
// @Success      200  {array}   common.BriefResponse
// @Failure      401  {object}  string "Unauthorized"
// @Failure      500  {object}  string "Internal error"
// @Security     BearerAuth
// @Router       /briefs/list [get]
func (s *Server) ListBriefs(c echo.Context) error {
	userID, err := s.GetContextUser(c)
	if err != nil {
		s.log.Errorf("get user id: %v", err)
		status := common.GetHTTPStatus(common.ErrUnauthorized)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrUnauthorized)})
	}

	briefs, err := s.customers.GetCustomerBriefs(c.Request().Context(), userID)
	if err != nil {
		s.log.Errorf("get customer briefs: %v", err)
		status := common.GetHTTPStatus(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInternalError)})
	}

	// Fetch full brief details if needed, currently briefs contains CustomerBrief edges
	// If we need the actual Brief objects, we need to load them.
	// The GetCustomerBriefs returns []*ent.CustomerBrief, which has Edges.Brief

	// Prepare response
	var result []*ent.Brief
	for _, cb := range briefs {
		if cb.Edges.Brief != nil {
			result = append(result, cb.Edges.Brief)
		}
	}

	return c.JSON(http.StatusOK, common.MapBriefs(result))
}

// UpdateBrief updates a brief
// @Summary      Update brief
// @Description  Update brief details
// @Tags         briefs
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Brief ID"
// @Success      204  {object}  nil
// @Failure      400  {object}  string "Invalid ID"
// @Security     BearerAuth
// @Router       /briefs/{id} [patch]
func (s *Server) UpdateBrief(c echo.Context) error {
	idStr := c.Param("id")
	_, err := uuid.Parse(idStr)
	if err != nil {
		s.log.Errorf("update brief: %v", err)
		status := common.GetHTTPStatus(common.ErrInvalidUUID)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidUUID)})
	}

	// var req UpdateBriefRequest
	// if err := c.Bind(&req); err != nil { ... }

	// brief, err := s.briefs.UpdateBrief(...)

	// Placeholder as Brief entity currently has no editable fields
	return c.JSON(http.StatusNoContent, "nothing to update")
}
