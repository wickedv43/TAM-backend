package server

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/wickedv43/TAM-backend/internal/database/ent"
)

type AdminClaims struct {
	ID         uuid.UUID `json:"uuid"`
	TelegramID int64     `json:"telegram_id"`
	Status     int       `json:"status"`
	jwt.RegisteredClaims
}

func (s *Server) generateJWT(c *ent.Customer) (string, error) {
	claims := AdminClaims{
		ID:         c.ID,
		TelegramID: c.TgID,
		Status:     c.Status,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.cfg.JWTExpire)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.cfg.JWTSecret)
}

func (s *Server) validateJWT(tokenString string) (*AdminClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AdminClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.cfg.JWTSecret, nil
	})
	if err != nil {
		return nil, errors.New("parse token")
	}

	if claims, ok := token.Claims.(*AdminClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
