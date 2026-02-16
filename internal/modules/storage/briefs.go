package storage

import (
	"context"

	"github.com/google/uuid"
	"github.com/wickedv43/TAM-backend/internal/database/ent"
	"github.com/wickedv43/TAM-backend/internal/database/ent/brief"
	"github.com/wickedv43/TAM-backend/internal/database/ent/customerbrief"
)

type BriefsInterface interface {
	CreateBrief(ctx context.Context, b *ent.Brief) (*ent.Brief, error)
	GetBrief(ctx context.Context, id uuid.UUID) (*ent.Brief, error)
	UpdateBrief(ctx context.Context, id uuid.UUID, updater func(*ent.BriefUpdateOne)) (*ent.Brief, error)
	DeleteBrief(ctx context.Context, id uuid.UUID) error

	GetBriefCustomers(ctx context.Context, briefID uuid.UUID) ([]*ent.CustomerBrief, error)
	GetBriefCustomerIDs(ctx context.Context, briefID uuid.UUID) ([]uuid.UUID, error)
	GetBriefCustomer(ctx context.Context, briefID, customerID uuid.UUID) (*ent.CustomerBrief, error)
	AddBriefCustomer(ctx context.Context, briefID, customerID uuid.UUID) (*ent.CustomerBrief, error)
	RemoveBriefCustomer(ctx context.Context, briefID, customerID uuid.UUID) error
}

// CreateBrief creates a new brief.
func (p *PostgresDB) CreateBrief(ctx context.Context, _ *ent.Brief) (*ent.Brief, error) {
	return p.db.Brief.Create().Save(ctx)
}

// GetBrief returns a brief by ID.
func (p *PostgresDB) GetBrief(ctx context.Context, id uuid.UUID) (*ent.Brief, error) {
	return p.db.Brief.Query().
		Where(brief.ID(id)).
		First(ctx)
}

// UpdateBrief updates a brief using the provided updater function.
func (p *PostgresDB) UpdateBrief(ctx context.Context, id uuid.UUID, updater func(*ent.BriefUpdateOne)) (*ent.Brief, error) {
	update := p.db.Brief.UpdateOneID(id)
	updater(update)
	return update.Save(ctx)
}

// DeleteBrief deletes a brief by ID.
func (p *PostgresDB) DeleteBrief(ctx context.Context, id uuid.UUID) error {
	_, err := p.db.Brief.Delete().Where(brief.ID(id)).Exec(ctx)
	return err
}

// GetBriefCustomers returns all customers linked to a brief.
func (p *PostgresDB) GetBriefCustomers(ctx context.Context, briefID uuid.UUID) ([]*ent.CustomerBrief, error) {
	return p.db.CustomerBrief.Query().
		Where(customerbrief.BriefID(briefID)).
		WithCustomer().
		All(ctx)
}

// GetBriefCustomerIDs returns only customer IDs linked to a brief.
func (p *PostgresDB) GetBriefCustomerIDs(ctx context.Context, briefID uuid.UUID) ([]uuid.UUID, error) {
	cbs, err := p.db.CustomerBrief.Query().
		Where(customerbrief.BriefID(briefID)).
		Select(customerbrief.FieldCustomerID).
		All(ctx)
	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, len(cbs))
	for i, cb := range cbs {
		ids[i] = cb.CustomerID
	}
	return ids, nil
}

// GetBriefCustomer returns the customer-brief link for the given IDs.
func (p *PostgresDB) GetBriefCustomer(ctx context.Context, briefID, customerID uuid.UUID) (*ent.CustomerBrief, error) {
	return p.db.CustomerBrief.Query().
		Where(
			customerbrief.BriefID(briefID),
			customerbrief.CustomerID(customerID),
		).
		Only(ctx)
}

// AddBriefCustomer adds a customer to a brief.
func (p *PostgresDB) AddBriefCustomer(ctx context.Context, briefID, customerID uuid.UUID) (*ent.CustomerBrief, error) {
	return p.db.CustomerBrief.Create().
		SetBriefID(briefID).
		SetCustomerID(customerID).
		Save(ctx)
}

// RemoveBriefCustomer removes a customer from a brief.
func (p *PostgresDB) RemoveBriefCustomer(ctx context.Context, briefID, customerID uuid.UUID) error {
	_, err := p.db.CustomerBrief.Delete().
		Where(
			customerbrief.BriefID(briefID),
			customerbrief.CustomerID(customerID),
		).
		Exec(ctx)
	return err
}
