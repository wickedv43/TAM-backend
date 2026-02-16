package storage

import (
	"context"

	"github.com/google/uuid"
	"github.com/wickedv43/TAM-backend/internal/common"
	"github.com/wickedv43/TAM-backend/internal/database/ent"
	"github.com/wickedv43/TAM-backend/internal/database/ent/customer"
	"github.com/wickedv43/TAM-backend/internal/database/ent/customerchannel"
)

type CustomersInterface interface {
	CreateCustomer(ctx context.Context, c *ent.Customer) (*ent.Customer, error)
	GetCustomer(ctx context.Context, uuid uuid.UUID) (*ent.Customer, error)
	GetCustomerByTgID(ctx context.Context, id int64) (*ent.Customer, error)
	IsExist(ctx context.Context, id int64) (bool, error)
	UpdateCustomer(ctx context.Context, id uuid.UUID, updater func(*ent.CustomerUpdateOne)) (*ent.Customer, error)
	DeleteCustomer(ctx context.Context, id uuid.UUID) error

	GetCustomerChannels(ctx context.Context, customerID uuid.UUID) ([]*ent.CustomerChannel, error)
	GetCustomerChannelIDs(ctx context.Context, customerID uuid.UUID) ([]uuid.UUID, error)
	GetCustomerChannel(ctx context.Context, customerID, channelID uuid.UUID) (*ent.CustomerChannel, error)
	AddCustomerChannel(ctx context.Context, customerID, channelID uuid.UUID, role int, permissions []int) (*ent.CustomerChannel, error)
	RemoveCustomerChannel(ctx context.Context, customerID, channelID uuid.UUID) error
	UpdateCustomerChannelRole(ctx context.Context, customerID, channelID uuid.UUID, role int, permissions []int) (*ent.CustomerChannel, error)

	GetCustomerBriefs(ctx context.Context, customerID uuid.UUID) ([]*ent.CustomerBrief, error)

	GetCustomerAdvertiserDeals(ctx context.Context, customerID uuid.UUID) ([]*ent.Deal, error)
	GetCustomerManagedDeals(ctx context.Context, customerID uuid.UUID) ([]*ent.Deal, error)

	LockBalanceForDeal(ctx context.Context, deal *ent.Deal) error
	UnlockBalanceForDeal(ctx context.Context, deal *ent.Deal) error
	TransferBalanceToManagerForDeal(ctx context.Context, deal *ent.Deal) error

	DeductBalance(ctx context.Context, customerID uuid.UUID, amount float64) error
	RefundBalance(ctx context.Context, customerID uuid.UUID, amount float64) error
}

func (p *PostgresDB) CreateCustomer(ctx context.Context, c *ent.Customer) (*ent.Customer, error) {
	created, err := p.db.Customer.Create().
		SetTgID(c.TgID).
		SetTgUsername(c.TgUsername).
		SetTgFirstname(c.TgFirstname).
		SetTgLastname(c.TgLastname).
		SetTgPicture(c.TgPicture).
		SetTgIsPremium(c.TgIsPremium).
		SetTgIsAllowPm(c.TgIsAllowPm).
		SetAddressBounceable(c.AddressBounceable).
		SetAddressNonbounceable(c.AddressNonbounceable).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return p.GetCustomer(ctx, created.ID)
}

func (p *PostgresDB) IsExist(ctx context.Context, id int64) (bool, error) {
	return p.db.Customer.Query().
		Where(customer.TgID(id)).
		Exist(ctx)
}

func (p *PostgresDB) GetCustomerByTgID(ctx context.Context, id int64) (*ent.Customer, error) {
	return p.db.Customer.Query().
		Where(customer.TgID(id)).
		First(ctx)
}

func (p *PostgresDB) GetCustomer(ctx context.Context, uID uuid.UUID) (*ent.Customer, error) {
	return p.db.Customer.Query().
		Where(customer.ID(uID)).
		First(ctx)
}

// GetCustomerChannels returns customer channels with role info.
func (p *PostgresDB) GetCustomerChannels(ctx context.Context, customerID uuid.UUID) ([]*ent.CustomerChannel, error) {
	return p.db.CustomerChannel.Query().
		Where(customerchannel.CustomerID(customerID)).
		WithChannel().
		All(ctx)
}

// GetCustomerChannelIDs returns only channel IDs for a customer.
func (p *PostgresDB) GetCustomerChannelIDs(ctx context.Context, customerID uuid.UUID) ([]uuid.UUID, error) {
	ccs, err := p.db.CustomerChannel.Query().
		Where(customerchannel.CustomerID(customerID)).
		Select(customerchannel.FieldChannelID).
		All(ctx)
	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, len(ccs))
	for i, cc := range ccs {
		ids[i] = cc.ChannelID
	}
	return ids, nil
}

// GetCustomerChannel returns the customer-channel link for the given IDs.
func (p *PostgresDB) GetCustomerChannel(ctx context.Context, customerID, channelID uuid.UUID) (*ent.CustomerChannel, error) {
	return p.db.CustomerChannel.Query().
		Where(
			customerchannel.CustomerID(customerID),
			customerchannel.ChannelID(channelID),
		).
		Only(ctx)
}

// RemoveCustomerChannel removes the customer from the channel.
func (p *PostgresDB) RemoveCustomerChannel(ctx context.Context, customerID, channelID uuid.UUID) error {
	_, err := p.db.CustomerChannel.Delete().
		Where(
			customerchannel.CustomerID(customerID),
			customerchannel.ChannelID(channelID),
		).
		Exec(ctx)
	return err
}

// AddCustomerChannel adds a customer to a channel with the given role.
func (p *PostgresDB) AddCustomerChannel(ctx context.Context, customerID, channelID uuid.UUID, role int, permissions []int) (*ent.CustomerChannel, error) {
	create := p.db.CustomerChannel.Create().
		SetCustomerID(customerID).
		SetChannelID(channelID).
		SetRole(role)

	if permissions != nil {
		create = create.SetPermissions(permissions)
	}

	return create.Save(ctx)
}

// UpdateCustomerChannelRole updates the customer's role and permissions in the channel.
func (p *PostgresDB) UpdateCustomerChannelRole(ctx context.Context, customerID, channelID uuid.UUID, role int, permissions []int) (*ent.CustomerChannel, error) {
	cc, err := p.db.CustomerChannel.Query().
		Where(
			customerchannel.CustomerID(customerID),
			customerchannel.ChannelID(channelID),
		).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	update := p.db.CustomerChannel.UpdateOneID(cc.ID).
		SetRole(role)

	if permissions != nil {
		update = update.SetPermissions(permissions)
	} else {
		update = update.ClearPermissions()
	}

	return update.Save(ctx)
}

// UpdateCustomer updates a customer using the provided updater function.
func (p *PostgresDB) UpdateCustomer(ctx context.Context, id uuid.UUID, updater func(*ent.CustomerUpdateOne)) (*ent.Customer, error) {
	update := p.db.Customer.UpdateOneID(id)
	updater(update)
	return update.Save(ctx)
}

// DeleteCustomer deletes a customer by ID.
func (p *PostgresDB) DeleteCustomer(ctx context.Context, id uuid.UUID) error {
	_, err := p.db.Customer.Delete().Where(customer.ID(id)).Exec(ctx)
	return err
}

// GetCustomerBriefs returns all briefs for a customer.
func (p *PostgresDB) GetCustomerBriefs(ctx context.Context, customerID uuid.UUID) ([]*ent.CustomerBrief, error) {
	cust, err := p.db.Customer.Query().
		Where(customer.ID(customerID)).
		WithCustomerBriefs().
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return cust.Edges.CustomerBriefs, nil
}

// GetCustomerAdvertiserDeals returns deals where the customer is the advertiser.
func (p *PostgresDB) GetCustomerAdvertiserDeals(ctx context.Context, customerID uuid.UUID) ([]*ent.Deal, error) {
	cust, err := p.db.Customer.Query().
		Where(customer.ID(customerID)).
		WithAdvertiserDeals().
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return cust.Edges.AdvertiserDeals, nil
}

// GetCustomerManagedDeals returns deals where the customer is the channel manager.
func (p *PostgresDB) GetCustomerManagedDeals(ctx context.Context, customerID uuid.UUID) ([]*ent.Deal, error) {
	cust, err := p.db.Customer.Query().
		Where(customer.ID(customerID)).
		WithManagedDeals().
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return cust.Edges.ManagedDeals, nil
}

func (p *PostgresDB) LockBalanceForDeal(ctx context.Context, deal *ent.Deal) error {
	amount := deal.TonPrice
	if amount <= 0 {
		return nil
	}
	affected, err := p.db.Customer.Update().
		Where(
			customer.ID(deal.AdvertiserCustomerID),
			customer.TonBalanceGTE(amount),
		).
		AddTonBalance(-amount).
		AddTonBalanceLocked(amount).
		Save(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return common.ErrInsufficientBalance
	}
	return nil
}

func (p *PostgresDB) UnlockBalanceForDeal(ctx context.Context, deal *ent.Deal) error {
	amount := deal.TonPrice
	if amount <= 0 {
		return nil
	}
	_, err := p.db.Customer.UpdateOneID(deal.AdvertiserCustomerID).
		AddTonBalance(amount).
		AddTonBalanceLocked(-amount).
		Save(ctx)
	return err
}

// TransferBalanceToManagerForDeal transfers locked balance from advertiser to channel manager on deal completion.
func (p *PostgresDB) TransferBalanceToManagerForDeal(ctx context.Context, deal *ent.Deal) error {
	amount := deal.TonPrice
	if amount <= 0 {
		return nil
	}

	tx, err := p.db.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if v := recover(); v != nil {
			_ = tx.Rollback()
		}
	}()

	_, err = tx.Customer.UpdateOneID(deal.AdvertiserCustomerID).
		AddTonBalanceLocked(-amount).
		Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	_, err = tx.Customer.UpdateOneID(deal.ChannelManagerID).
		AddTonBalance(amount).
		Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

// DeductBalance atomically subtracts amount from customer's ton_balance.
// Returns ErrInsufficientBalance if ton_balance < amount.
func (p *PostgresDB) DeductBalance(ctx context.Context, customerID uuid.UUID, amount float64) error {
	if amount <= 0 {
		return nil
	}
	affected, err := p.db.Customer.Update().
		Where(
			customer.ID(customerID),
			customer.TonBalanceGTE(amount),
		).
		AddTonBalance(-amount).
		Save(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return common.ErrInsufficientBalance
	}
	return nil
}

// RefundBalance adds amount back to customer's ton_balance (e.g. after failed withdrawal send).
func (p *PostgresDB) RefundBalance(ctx context.Context, customerID uuid.UUID, amount float64) error {
	if amount <= 0 {
		return nil
	}
	_, err := p.db.Customer.UpdateOneID(customerID).
		AddTonBalance(amount).
		Save(ctx)
	return err
}
