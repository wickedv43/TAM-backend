package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/wickedv43/TAM-backend/internal/common"
	"github.com/wickedv43/TAM-backend/internal/database/ent"
)

// GetMe returns the current user profile
// @Summary      Get current user
// @Description  Get profile of logged-in user
// @Tags         customers
// @Accept       json
// @Produce      json
// @Success      200  {object}  common.CustomerResponse
// @Failure      401  {object}  string "Unauthorized"
// @Failure      404  {object}  string "User not found"
// @Failure      500  {object}  string "Internal error"
// @Security     BearerAuth
// @Router       /customers/me [get]
func (s *Server) GetMe(c echo.Context) error {
	userID, err := s.GetContextUser(c)
	if err != nil {
		s.log.Errorf("invalid user id: %v", err)
		status := common.GetHTTPStatus(common.ErrUnauthorized)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrUnauthorized)})
	}

	customer, err := s.customers.GetCustomer(c.Request().Context(), userID)
	if err != nil {
		if ent.IsNotFound(err) {
			s.log.Errorf("customer not found: %v", err)
			status := common.GetHTTPStatus(common.ErrNotFound)
			return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrNotFound)})
		}
		s.log.Errorf("get customer: %v", err)
		status := common.GetHTTPStatus(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInternalError)})
	}

	return c.JSON(http.StatusOK, common.MapCustomer(customer))
}

// UpdateMe updates the current user profile
// @Summary      Update user
// @Description  Update profile details
// @Tags         customers
// @Accept       json
// @Produce      json
// @Param        input body common.UpdateMeRequest true "Update data"
// @Success      200  {object}  common.CustomerResponse
// @Failure      400  {object}  string "Invalid request"
// @Failure      401  {object}  string "Unauthorized"
// @Failure      500  {object}  string "Internal error"
// @Security     BearerAuth
// @Router       /customers/me [patch]
func (s *Server) UpdateMe(c echo.Context) error {
	userID, err := s.GetContextUser(c)
	if err != nil {
		s.log.Errorf("invalid user id: %v", err)
		status := common.GetHTTPStatus(common.ErrUnauthorized)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrUnauthorized)})
	}

	var req common.UpdateMeRequest
	if err = c.Bind(&req); err != nil {
		s.log.Errorf("invalid body: %v", err)
		status := common.GetHTTPStatus(common.ErrInvalidRequest)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInvalidRequest)})
	}

	customer, err := s.customers.UpdateCustomer(c.Request().Context(), userID, func(u *ent.CustomerUpdateOne) {
		if req.TgUsername != nil {
			u.SetTgUsername(*req.TgUsername)
		}
		if req.TgFirstname != nil {
			u.SetTgFirstname(*req.TgFirstname)
		}
		if req.TgLastname != nil {
			u.SetTgLastname(*req.TgLastname)
		}
		if req.TgLanguage != nil {
			u.SetTgLanguage(*req.TgLanguage)
		}
		if req.TgPicture != nil {
			u.SetTgPicture(*req.TgPicture)
		}
		if req.AddressBounceable != nil {
			u.SetAddressBounceable(*req.AddressBounceable)
		}
		if req.AddressNonbounceable != nil {
			u.SetAddressNonbounceable(*req.AddressNonbounceable)
		}
	})

	if err != nil {
		s.log.Errorf("update customer: %v", err)
		status := common.GetHTTPStatus(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInternalError)})
	}

	return c.JSON(http.StatusOK, common.MapCustomer(customer))
}

// GetMyDeals returns deals related to the current user (as advertiser or manager)
// @Summary      Get user deals
// @Description  Get deals where user is advertiser or manager
// @Tags         customers
// @Accept       json
// @Produce      json
// @Success      200  {object}  common.GetMyDealsResponse
// @Failure      401  {object}  string "Unauthorized"
// @Failure      500  {object}  string "Internal error"
// @Security     BearerAuth
// @Router       /customers/me/deals [get]
func (s *Server) GetMyDeals(c echo.Context) error {
	userID, err := s.GetContextUser(c)
	if err != nil {
		s.log.Errorf("invalid user id: %v", err)
		status := common.GetHTTPStatus(common.ErrUnauthorized)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrUnauthorized)})
	}

	// Get deals where user is advertiser
	advertiserDeals, err := s.customers.GetCustomerAdvertiserDeals(c.Request().Context(), userID)
	if err != nil {
		s.log.Errorf("get advertiser deals: %v", err)
		status := common.GetHTTPStatus(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInternalError)})
	}

	// Get deals where user is manager
	managerDeals, err := s.customers.GetCustomerManagedDeals(c.Request().Context(), userID)
	if err != nil {
		s.log.Errorf("get manager deals: %v", err)
		status := common.GetHTTPStatus(common.ErrInternalError)
		return c.JSON(status, map[string]string{"error": s.GetUserMessage(common.ErrInternalError)})
	}

	return c.JSON(http.StatusOK, common.GetMyDealsResponse{
		AdvertiserDeals: common.MapDeals(advertiserDeals),
		ManagerDeals:    common.MapDeals(managerDeals),
	})
}
