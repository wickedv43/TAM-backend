package storage

import (
	"context"

	"github.com/google/uuid"
	"github.com/wickedv43/TAM-backend/internal/common"
	"github.com/wickedv43/TAM-backend/internal/common/constants/customer_channel_role"
	"github.com/wickedv43/TAM-backend/internal/database/ent"
	"github.com/wickedv43/TAM-backend/internal/database/ent/channel"
	"github.com/wickedv43/TAM-backend/internal/database/ent/customerchannel"
)

type ChannelsInterface interface {
	CreateChannel(ctx context.Context, c *ent.Channel) (*ent.Channel, error)
	GetChannel(ctx context.Context, id uuid.UUID) (*ent.Channel, error)
	GetChannelWithCustomerChannels(ctx context.Context, id uuid.UUID) (*ent.Channel, error)
	GetChannelByTgID(ctx context.Context, tgID int64) (*ent.Channel, error)
	IsChannelExist(ctx context.Context, tgID int64) (bool, error)
	UpdateChannel(ctx context.Context, id uuid.UUID, updater func(*ent.ChannelUpdateOne)) (*ent.Channel, error)
	DeleteChannel(ctx context.Context, id uuid.UUID) error

	GetAllChannels(ctx context.Context) ([]*ent.Channel, error)
	GetChannelsByStatus(ctx context.Context, status int) ([]*ent.Channel, error)
	GetChannelsByIsListed(ctx context.Context, isListed bool) ([]*ent.Channel, error)
	GetChannelsByMainLanguage(ctx context.Context, language string) ([]*ent.Channel, error)
	GetListedChannels(ctx context.Context) ([]*ent.Channel, error)
	GetChannelsByStatusAndIsListed(ctx context.Context, status int, isListed bool) ([]*ent.Channel, error)
	GetListedChannelsPaginated(ctx context.Context, pageOpts common.PageOptions) (*common.Pagination[[]*ent.Channel], error)
	GetCustomerChannelsPaginated(ctx context.Context, customerID uuid.UUID, pageOpts common.PageOptions) (*common.Pagination[[]*ent.Channel], error)

	GetChannelCustomers(ctx context.Context, channelID uuid.UUID) ([]*ent.CustomerChannel, error)
	GetChannelCustomerIDs(ctx context.Context, channelID uuid.UUID) ([]uuid.UUID, error)
	GetChannelCustomer(ctx context.Context, channelID, customerID uuid.UUID) (*ent.CustomerChannel, error)

	GetChannelDeals(ctx context.Context, channelID uuid.UUID) ([]*ent.Deal, error)
}

// GetAllChannels returns all channels.
func (p *PostgresDB) GetAllChannels(ctx context.Context) ([]*ent.Channel, error) {
	return p.db.Channel.Query().All(ctx)
}

// GetChannelsByStatus returns channels with the given status.
func (p *PostgresDB) GetChannelsByStatus(ctx context.Context, status int) ([]*ent.Channel, error) {
	return p.db.Channel.Query().
		Where(channel.Status(status)).
		All(ctx)
}

// GetChannelsByIsListed returns channels filtered by is_listed flag.
func (p *PostgresDB) GetChannelsByIsListed(ctx context.Context, isListed bool) ([]*ent.Channel, error) {
	return p.db.Channel.Query().
		Where(channel.IsListed(isListed)).
		All(ctx)
}

// GetChannelsByMainLanguage returns channels with the given main language.
func (p *PostgresDB) GetChannelsByMainLanguage(ctx context.Context, language string) ([]*ent.Channel, error) {
	return p.db.Channel.Query().
		Where(channel.MainLanguage(language)).
		All(ctx)
}

// GetListedChannels returns only listed channels (is_listed = true).
func (p *PostgresDB) GetListedChannels(ctx context.Context) ([]*ent.Channel, error) {
	return p.db.Channel.Query().
		Where(channel.IsListed(true)).
		All(ctx)
}

// GetChannelsByStatusAndIsListed returns channels filtered by status and is_listed.
func (p *PostgresDB) GetChannelsByStatusAndIsListed(ctx context.Context, status int, isListed bool) ([]*ent.Channel, error) {
	return p.db.Channel.Query().
		Where(
			channel.Status(status),
			channel.IsListed(isListed),
		).
		All(ctx)
}

// CreateChannel creates a new channel.
func (p *PostgresDB) CreateChannel(ctx context.Context, c *ent.Channel) (*ent.Channel, error) {
	create := p.db.Channel.Create().
		SetTgID(c.TgID).
		SetIsListed(c.IsListed).
		SetPrices(c.Prices).
		SetStatus(c.Status)

	if c.TgUsername != "" {
		create = create.SetTgUsername(c.TgUsername)
	}
	if c.TgName != "" {
		create = create.SetTgName(c.TgName)
	}
	if c.TgDescription != "" {
		create = create.SetTgDescription(c.TgDescription)
	}
	if c.TgPicture != "" {
		create = create.SetTgPicture(c.TgPicture)
	}
	if c.Tags != nil {
		create = create.SetTags(c.Tags)
	}
	if c.Stats != nil {
		create = create.SetStats(c.Stats)
	}
	if c.Subscribers != 0 {
		create = create.SetNillableSubscribers(&c.Subscribers)
	}
	if c.PremiumSubscribers != 0 {
		create = create.SetNillablePremiumSubscribers(&c.PremiumSubscribers)
	}
	if c.MedianPostViews != 0 {
		create = create.SetNillableMedianPostViews(&c.MedianPostViews)
	}
	if c.AvgPostViews != 0 {
		create = create.SetNillableAvgPostViews(&c.AvgPostViews)
	}
	if c.MainLanguage != "" {
		create = create.SetMainLanguage(c.MainLanguage)
	}
	if c.NotificationsOn != 0 {
		create = create.SetNotificationsOn(c.NotificationsOn)
	}

	if !c.FirstPostDate.IsZero() {
		create = create.SetFirstPostDate(c.FirstPostDate)
	}

	if c.TotalPosts != 0 {
		create = create.SetTotalPosts(c.TotalPosts)
	}

	return create.Save(ctx)
}

// GetChannel returns a channel by ID.
func (p *PostgresDB) GetChannel(ctx context.Context, id uuid.UUID) (*ent.Channel, error) {
	return p.db.Channel.Query().
		Where(channel.ID(id)).
		First(ctx)
}

func (p *PostgresDB) GetChannelWithCustomerChannels(ctx context.Context, id uuid.UUID) (*ent.Channel, error) {
	return p.db.Channel.Query().
		Where(channel.ID(id)).
		WithCustomerChannels().
		Only(ctx)
}

// GetChannelByTgID returns a channel by Telegram ID.
func (p *PostgresDB) GetChannelByTgID(ctx context.Context, tgID int64) (*ent.Channel, error) {
	return p.db.Channel.Query().
		Where(channel.TgID(tgID)).
		First(ctx)
}

// IsChannelExist checks if a channel exists by Telegram ID.
func (p *PostgresDB) IsChannelExist(ctx context.Context, tgID int64) (bool, error) {
	return p.db.Channel.Query().
		Where(channel.TgID(tgID)).
		Exist(ctx)
}

// UpdateChannel updates a channel using the provided updater function.
func (p *PostgresDB) UpdateChannel(ctx context.Context, id uuid.UUID, updater func(*ent.ChannelUpdateOne)) (*ent.Channel, error) {
	update := p.db.Channel.UpdateOneID(id)
	updater(update)
	return update.Save(ctx)
}

// DeleteChannel deletes a channel by ID.
func (p *PostgresDB) DeleteChannel(ctx context.Context, id uuid.UUID) error {
	_, err := p.db.Channel.Delete().Where(channel.ID(id)).Exec(ctx)
	return err
}

// GetChannelCustomers returns all customers of a channel with role info.
func (p *PostgresDB) GetChannelCustomers(ctx context.Context, channelID uuid.UUID) ([]*ent.CustomerChannel, error) {
	return p.db.CustomerChannel.Query().
		Where(customerchannel.ChannelID(channelID)).
		WithCustomer().
		All(ctx)
}

// GetChannelCustomerIDs returns only customer IDs for a channel.
func (p *PostgresDB) GetChannelCustomerIDs(ctx context.Context, channelID uuid.UUID) ([]uuid.UUID, error) {
	ccs, err := p.db.CustomerChannel.Query().
		Where(customerchannel.ChannelID(channelID)).
		Select(customerchannel.FieldCustomerID).
		All(ctx)
	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, len(ccs))
	for i, cc := range ccs {
		ids[i] = cc.CustomerID
	}
	return ids, nil
}

// GetChannelCustomer returns the customer-channel link for the given IDs.
func (p *PostgresDB) GetChannelCustomer(ctx context.Context, channelID, customerID uuid.UUID) (*ent.CustomerChannel, error) {
	return p.db.CustomerChannel.Query().
		Where(
			customerchannel.ChannelID(channelID),
			customerchannel.CustomerID(customerID),
		).
		Only(ctx)
}

// GetChannelDeals returns all deals for a channel.
func (p *PostgresDB) GetChannelDeals(ctx context.Context, channelID uuid.UUID) ([]*ent.Deal, error) {
	ch, err := p.db.Channel.Query().
		Where(channel.ID(channelID)).
		WithDeals().
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return ch.Edges.Deals, nil
}

func (p *PostgresDB) GetListedChannelsPaginated(ctx context.Context, pageOpts common.PageOptions) (*common.Pagination[[]*ent.Channel], error) {
	queryBuilder := p.db.Channel.Query().
		Where(channel.IsListed(true))

	if pageOpts.Order == "ASC" {
		queryBuilder = queryBuilder.Order(ent.Asc(channel.FieldCreatedAt))
	} else {
		queryBuilder = queryBuilder.Order(ent.Desc(channel.FieldCreatedAt))
	}

	return common.Paginate(ctx, queryBuilder, pageOpts)
}

func (p *PostgresDB) GetCustomerChannelsPaginated(ctx context.Context, customerID uuid.UUID, pageOpts common.PageOptions) (*common.Pagination[[]*ent.Channel], error) {
	queryBuilder := p.db.Channel.Query().
		Where(channel.HasCustomerChannelsWith(
			customerchannel.CustomerID(customerID),
			customerchannel.RoleIn(
				int(customer_channel_role.ADMIN),
				int(customer_channel_role.OWNER),
			),
		))

	if pageOpts.Order == "ASC" {
		queryBuilder = queryBuilder.Order(ent.Asc(channel.FieldCreatedAt))
	} else {
		queryBuilder = queryBuilder.Order(ent.Desc(channel.FieldCreatedAt))
	}

	return common.Paginate(ctx, queryBuilder, pageOpts)
}

// CountListedChannels returns the count of listed channels.
func (p *PostgresDB) CountListedChannels(ctx context.Context) (int, error) {
	return p.db.Channel.Query().
		Where(channel.IsListed(true)).
		Count(ctx)
}
