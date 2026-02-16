package common

import (
	"time"

	"github.com/google/uuid"
	"github.com/wickedv43/TAM-backend/internal/common/constants/customer_status"
	"github.com/wickedv43/TAM-backend/internal/database/ent"
)

// --- Customer ---
type UpdateMeRequest struct {
	TgUsername           *string `json:"tg_username"`
	TgFirstname          *string `json:"tg_firstname"`
	TgLastname           *string `json:"tg_lastname"`
	TgLanguage           *string `json:"tg_language"`
	TgPicture            *string `json:"tg_picture"`
	AddressBounceable    *string `json:"address_bounceable"`
	AddressNonbounceable *string `json:"address_nonbounceable"`
}

type WithdrawRequest struct {
	Amount    float64 `json:"amount"`     // amount in TON
	ToAddress string  `json:"to_address"` // destination TON address
}

type WithdrawResponse struct {
	TxHash string `json:"tx_hash"` // transaction hash (base64)
}

type CustomerResponse struct {
	ID                   uuid.UUID                      `json:"id"`
	TgID                 int64                          `json:"tg_id"`
	TgUsername           string                         `json:"tg_username"`
	TgFirstname          string                         `json:"tg_firstname"`
	TgLastname           string                         `json:"tg_lastname"`
	TgLanguage           string                         `json:"tg_language"`
	TgPicture            string                         `json:"tg_picture"`
	TgIsPremium          bool                           `json:"tg_is_premium"`
	AddressBounceable    string                         `json:"address_bounceable"`
	AddressNonbounceable string                         `json:"address_nonbounceable"`
	TonBalance           float64                        `json:"ton_balance"`
	TonBalanceLocked     float64                        `json:"ton_balance_locked"`
	Status               customer_status.CustomerStatus `json:"status"`
	CreatedAt            time.Time                      `json:"created_at"`
	UpdatedAt            time.Time                      `json:"updated_at"`
}

func MapCustomer(c *ent.Customer) *CustomerResponse {
	if c == nil {
		return nil
	}
	return &CustomerResponse{
		ID:                   c.ID,
		TgID:                 c.TgID,
		TgUsername:           c.TgUsername,
		TgFirstname:          c.TgFirstname,
		TgLastname:           c.TgLastname,
		TgLanguage:           c.TgLanguage,
		TgPicture:            c.TgPicture,
		TgIsPremium:          c.TgIsPremium,
		AddressBounceable:    c.AddressBounceable,
		AddressNonbounceable: c.AddressNonbounceable,
		TonBalance:           c.TonBalance,
		TonBalanceLocked:     c.TonBalanceLocked,
		Status:               customer_status.CustomerStatus(c.Status),
		CreatedAt:            c.CreatedAt,
		UpdatedAt:            c.UpdatedAt,
	}
}
