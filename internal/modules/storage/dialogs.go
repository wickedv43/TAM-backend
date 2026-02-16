package storage

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/wickedv43/TAM-backend/internal/common/constants/bot_state"
	"github.com/wickedv43/TAM-backend/internal/database/ent"
	"github.com/wickedv43/TAM-backend/internal/database/ent/botstate"
)

const (
	timeout = 15 * time.Second
)

type BotStateInterface interface {
	SetState(userID int64, state bot_state.BotState, dealID uuid.UUID, data map[string]interface{}) (*ent.BotState, error)
	GetState(userID int64) (*ent.BotState, error)
	ClearState(userID int64) error
}

// SetState upserts the user state
func (p *PostgresDB) SetState(userID int64, state bot_state.BotState, dealID uuid.UUID, data map[string]interface{}) (*ent.BotState, error) {
	ctx, cancel := context.WithTimeout(p.rootCtx, timeout)
	defer cancel()

	err := p.db.BotState.Create().
		SetUserID(userID).
		SetState(int(state)).
		SetDealID(dealID).
		SetData(data).
		OnConflictColumns(botstate.FieldUserID).
		UpdateNewValues().
		Exec(ctx)

	if err != nil {
		return nil, err
	}

	return p.db.BotState.Query().
		Where(botstate.UserID(userID)).
		Only(ctx)
}

// GetState returns the current state or nil if not found
func (p *PostgresDB) GetState(userID int64) (*ent.BotState, error) {
	ctx, cancel := context.WithTimeout(p.rootCtx, timeout)
	defer cancel()

	s, err := p.db.BotState.Query().
		Where(botstate.UserID(userID)).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return s, nil
}

// ClearState deletes the state entry for the user
func (p *PostgresDB) ClearState(userID int64) error {
	ctx, cancel := context.WithTimeout(p.rootCtx, timeout)
	defer cancel()

	_, err := p.db.BotState.Delete().
		Where(botstate.UserID(userID)).
		Exec(ctx)
	return err
}
