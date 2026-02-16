package storage

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/wickedv43/TAM-backend/internal/common"
	"github.com/wickedv43/TAM-backend/internal/common/constants/deal_status"
	"github.com/wickedv43/TAM-backend/internal/database/ent"
	"github.com/wickedv43/TAM-backend/internal/database/ent/deal"
)

type DealsInterface interface {
	CreateDeal(ctx context.Context, d *ent.Deal) (*ent.Deal, error)
	GetDeal(ctx context.Context, id uuid.UUID) (*ent.Deal, error)
	UpdateDeal(ctx context.Context, id uuid.UUID, updater func(*ent.DealUpdateOne)) (*ent.Deal, error)
	DeleteDeal(ctx context.Context, id uuid.UUID) error

	GetDealsByChannel(ctx context.Context, channelID uuid.UUID) ([]*ent.Deal, error)
	GetDealsByChannelExcludingStatuses(ctx context.Context, channelID uuid.UUID) ([]*ent.Deal, error)
	GetDealsByAdvertiser(ctx context.Context, advertiserID uuid.UUID) ([]*ent.Deal, error)
	GetDealsByChannelManager(ctx context.Context, managerID uuid.UUID) ([]*ent.Deal, error)
	GetDealsByStatus(ctx context.Context, status int) ([]*ent.Deal, error)
	GetDealsByType(ctx context.Context, dealType int) ([]*ent.Deal, error)
	GetDealsByChannelAndStatus(ctx context.Context, channelID uuid.UUID, status int) ([]*ent.Deal, error)
	GetPublishedDealsWithinTopHoursByChannel(ctx context.Context, channelID uuid.UUID) ([]*ent.Deal, error)
	GetExpiredDeals(ctx context.Context) ([]*ent.Deal, error)

	GetDealsForRefund(ctx context.Context) ([]*ent.Deal, error)
	GetDealsNeedToComplete(ctx context.Context) ([]*ent.Deal, error)
	GetDealsPublishedActive(ctx context.Context) ([]*ent.Deal, error)
	GetUserDealsPaginated(ctx context.Context, userID uuid.UUID, pageOpts common.PageOptions) (*common.Pagination[[]*ent.Deal], error)
}

// CreateDeal creates a new deal.
func (p *PostgresDB) CreateDeal(ctx context.Context, d *ent.Deal) (*ent.Deal, error) {
	create := p.db.Deal.Create().
		SetChannelID(d.ChannelID).
		SetAdvertiserCustomerID(d.AdvertiserCustomerID).
		SetChannelManagerID(d.ChannelManagerID).
		SetType(d.Type).
		SetStatus(d.Status).
		SetStatusUpdatedAt(d.StatusUpdatedAt).
		SetExpiresAt(d.ExpiresAt).
		SetTargetType(d.TargetType).
		SetTonPrice(d.TonPrice)

	if len(d.ChannelPostIds) > 0 {
		create = create.SetChannelPostIds(d.ChannelPostIds)
	}

	if d.Target != nil {
		create = create.SetTarget(d.Target)
	}
	if d.PublicationTime != nil {
		create = create.SetNillablePublicationTime(d.PublicationTime)
	}

	return create.Save(ctx)
}

// GetDeal fetches a deal by ID with advertiser, channel and channel_manager edges.
func (p *PostgresDB) GetDeal(ctx context.Context, id uuid.UUID) (*ent.Deal, error) {
	return p.db.Deal.Query().
		Where(deal.ID(id)).
		WithAdvertiser().
		WithChannel().
		WithChannelManager().
		First(ctx)
}

// GetDealsByChannelExcludingStatuses returns channel deals whose status is not in the excluded set.
func (p *PostgresDB) GetDealsByChannelExcludingStatuses(ctx context.Context, channelID uuid.UUID) ([]*ent.Deal, error) {
	excludedStatuses := []int{
		int(deal_status.COMPLETED),
		int(deal_status.CANCELED),
		int(deal_status.EXPIRED),
		int(deal_status.TERMS_VIOLATED),
		int(deal_status.ERR_PUBLISH),
		int(deal_status.ERR_BAD_TARGET),
	}

	return p.db.Deal.Query().
		Where(
			deal.ChannelID(channelID),
			deal.StatusNotIn(excludedStatuses...),
		).
		All(ctx)
}

func (p *PostgresDB) GetDealsForRefund(ctx context.Context) ([]*ent.Deal, error) {
	var terminalRefundStatuses = []int{
		int(deal_status.CANCELED),
		int(deal_status.EXPIRED),
		int(deal_status.TERMS_VIOLATED),
		int(deal_status.ERR_PUBLISH),
		int(deal_status.ERR_BAD_TARGET),
	}

	return p.db.Deal.Query().
		Where(
			deal.StatusIn(terminalRefundStatuses...),
			deal.BalanceRefunded(false),
		).
		WithAdvertiser().
		All(ctx)
}

// UpdateDeal updates a deal using the provided updater function.
func (p *PostgresDB) UpdateDeal(ctx context.Context, id uuid.UUID, updater func(*ent.DealUpdateOne)) (*ent.Deal, error) {
	update := p.db.Deal.UpdateOneID(id)
	updater(update)
	return update.Save(ctx)
}

// DeleteDeal deletes a deal by ID.
func (p *PostgresDB) DeleteDeal(ctx context.Context, id uuid.UUID) error {
	_, err := p.db.Deal.Delete().Where(deal.ID(id)).Exec(ctx)
	return err
}

// GetDealsByChannel returns all deals for a channel.
func (p *PostgresDB) GetDealsByChannel(ctx context.Context, channelID uuid.UUID) ([]*ent.Deal, error) {
	return p.db.Deal.Query().
		Where(deal.ChannelID(channelID)).
		All(ctx)
}

// GetDealsByAdvertiser returns all deals where the user is the advertiser.
func (p *PostgresDB) GetDealsByAdvertiser(ctx context.Context, advertiserID uuid.UUID) ([]*ent.Deal, error) {
	return p.db.Deal.Query().
		Where(deal.AdvertiserCustomerID(advertiserID)).
		All(ctx)
}

// GetDealsByChannelManager returns all deals where the user is the channel manager.
func (p *PostgresDB) GetDealsByChannelManager(ctx context.Context, managerID uuid.UUID) ([]*ent.Deal, error) {
	return p.db.Deal.Query().
		Where(deal.ChannelManagerID(managerID)).
		All(ctx)
}

// GetDealsByStatus returns all deals with the given status.
func (p *PostgresDB) GetDealsByStatus(ctx context.Context, status int) ([]*ent.Deal, error) {
	return p.db.Deal.Query().
		Where(deal.Status(status)).
		All(ctx)
}

// GetDealsByType returns all deals of the given type.
func (p *PostgresDB) GetDealsByType(ctx context.Context, dealType int) ([]*ent.Deal, error) {
	return p.db.Deal.Query().
		Where(deal.Type(dealType)).
		All(ctx)
}

// GetDealsByChannelAndStatus returns channel deals with the given status.
func (p *PostgresDB) GetDealsByChannelAndStatus(ctx context.Context, channelID uuid.UUID, status int) ([]*ent.Deal, error) {
	return p.db.Deal.Query().
		Where(
			deal.ChannelID(channelID),
			deal.Status(status),
		).
		WithAdvertiser().
		WithChannelManager().
		WithChannel().
		All(ctx)
}

func (p *PostgresDB) GetPublishedDealsWithinTopHoursByChannel(ctx context.Context, channelID uuid.UUID) ([]*ent.Deal, error) {
	return p.db.Deal.Query().
		Where(
			deal.ChannelID(channelID),
			deal.Status(int(deal_status.PUBLISHED)),
			deal.TopDeadlineGT(time.Now()),
		).
		WithAdvertiser().
		WithChannelManager().
		WithChannel().
		All(ctx)
}

func (p *PostgresDB) GetDealsNeedToComplete(ctx context.Context) ([]*ent.Deal, error) {
	return p.db.Deal.Query().
		Where(
			deal.Status(int(deal_status.PUBLISHED)),
			deal.CommonDeadlineLTE(time.Now()),
		).
		WithAdvertiser().
		WithChannelManager().
		WithChannel().
		All(ctx)
}

// GetExpiredDeals returns all deals that have expired and are not in a terminal status.
func (p *PostgresDB) GetExpiredDeals(ctx context.Context) ([]*ent.Deal, error) {
	return p.db.Deal.Query().
		Where(
			deal.ExpiresAtLT(time.Now()),
			deal.StatusNEQ(int(deal_status.PUBLISHED)),
			deal.StatusNotIn(
				int(deal_status.CANCELED),
				int(deal_status.EXPIRED),
				int(deal_status.TERMS_VIOLATED),
				int(deal_status.COMPLETED),
			),
		).
		All(ctx)
}

func (p *PostgresDB) GetDealsPublishedActive(ctx context.Context) ([]*ent.Deal, error) {
	return p.db.Deal.Query().
		Where(
			deal.Status(int(deal_status.PUBLISHED)),
			deal.CommonDeadlineGT(time.Now()),
			deal.ChannelPostIdsNotNil(),
		).
		WithChannel().
		WithAdvertiser().
		WithChannelManager().
		All(ctx)
}

func (p *PostgresDB) GetUserDealsPaginated(ctx context.Context, userID uuid.UUID, pageOpts common.PageOptions) (*common.Pagination[[]*ent.Deal], error) {
	queryBuilder := p.db.Deal.Query().
		Where(
			deal.Or(
				deal.AdvertiserCustomerID(userID),
				deal.ChannelManagerID(userID),
			),
		).
		WithAdvertiser().
		WithChannel().
		WithChannelManager()

	if pageOpts.Order == "ASC" {
		queryBuilder = queryBuilder.Order(ent.Asc(deal.FieldCreatedAt))
	} else {
		queryBuilder = queryBuilder.Order(ent.Desc(deal.FieldCreatedAt))
	}

	return common.Paginate(ctx, queryBuilder, pageOpts)
}
