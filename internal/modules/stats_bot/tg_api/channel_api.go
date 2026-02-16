package tg_api

import (
	"context"
	"fmt"

	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
)

// AdminInfo holds admin permission data for a channel participant.
type AdminInfo struct {
	UserID    int64
	IsCreator bool
	CanPost   bool
	CanEdit   bool
	CanDelete bool
}

func ValidateChannelType(channel *tg.Channel) bool {
	if channel.Gigagroup {
		return false
	}
	if channel.Forum {
		return false
	}
	if channel.Megagroup {
		return false
	}
	if channel.Broadcast {
		return true
	}
	return true
}

func ValidateChannelIsPublic(channel *tg.Channel) bool {
	return channel.Username != ""
}

// ResolveChannelUsername converts channel username to InputChannel.
func (c *Client) ResolveChannelUsername(ctx context.Context, username string) (*tg.InputChannel, error) {
	resolved, err := c.api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: username,
	})
	if err != nil {
		return nil, fmt.Errorf("resolve username: %w", err)
	}

	for _, chat := range resolved.Chats {
		if channel, ok := chat.(*tg.Channel); ok {
			return &tg.InputChannel{
				ChannelID:  channel.ID,
				AccessHash: channel.AccessHash,
			}, nil
		}
	}

	return nil, fmt.Errorf("channel not found in results")
}

// GetFullChannel fetches full channel info.
func (c *Client) GetFullChannel(ctx context.Context, channel *tg.InputChannel) (*tg.MessagesChatFull, error) {
	return c.api.ChannelsGetFullChannel(ctx, channel)
}

// GetChannelHistory fetches channel message history.
func (c *Client) GetChannelHistory(ctx context.Context, channel *tg.InputChannel, limit int, offsetID int, addOffset int) ([]tg.MessageClass, int, error) {
	result, err := c.api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer: &tg.InputPeerChannel{
			ChannelID:  channel.ChannelID,
			AccessHash: channel.AccessHash,
		},
		Limit:      limit,
		OffsetID:   offsetID,
		OffsetDate: 0,
		AddOffset:  addOffset,
		MaxID:      0,
		MinID:      0,
		Hash:       0,
	})
	if err != nil {
		return nil, 0, err
	}

	switch m := result.(type) {
	case *tg.MessagesChannelMessages:
		return m.Messages, m.Count, nil
	case *tg.MessagesMessages:
		return m.Messages, len(m.Messages), nil
	case *tg.MessagesMessagesSlice:
		return m.Messages, m.Count, nil
	default:
		return nil, 0, fmt.Errorf("unexpected messages type: %T", result)
	}
}

// GetBoostsStatus fetches channel boosts status.
func (c *Client) GetBoostsStatus(ctx context.Context, channel *tg.InputChannel) (*tg.PremiumBoostsStatus, error) {
	peer := &tg.InputPeerChannel{
		ChannelID:  channel.ChannelID,
		AccessHash: channel.AccessHash,
	}
	return c.api.PremiumGetBoostsStatus(ctx, peer)
}

// GetBroadcastStats fetches channel stats with automatic DC migration handling.
func (c *Client) GetBroadcastStats(ctx context.Context, channel *tg.InputChannel, dark bool) (*tg.StatsBroadcastStats, int, error) {

	stats, err := c.api.StatsGetBroadcastStats(ctx, &tg.StatsGetBroadcastStatsRequest{
		Channel: channel,
		Dark:    dark,
	})

	if err == nil {
		return stats, 0, nil
	}

	if rpcErr, ok := tgerr.As(err); ok && rpcErr.Type == "STATS_MIGRATE" {
		targetDC := rpcErr.Argument
		c.log.Infof("Graphs located on DC%d, migrating...", targetDC)

		dcClient, err := c.client.GetDCClient(ctx, targetDC)
		if err != nil {
			return nil, 0, fmt.Errorf("create DC%d client: %w", targetDC, err)
		}

		stats, err = dcClient.API().StatsGetBroadcastStats(ctx, &tg.StatsGetBroadcastStatsRequest{
			Channel: channel,
			Dark:    dark,
		})
		if err != nil {
			return nil, 0, fmt.Errorf("get stats from DC%d: %w", targetDC, err)
		}

		c.log.Infof("Successfully got stats from DC%d", targetDC)
		return stats, targetDC, nil
	}

	return nil, 0, err
}

func (c *Client) LoadAsyncGraph(ctx context.Context, token string, dcID int) (string, error) {
	var result tg.StatsGraphClass
	var err error

	if dcID > 0 {
		dcClient, err := c.client.GetDCClient(ctx, dcID)
		if err != nil {
			return "", fmt.Errorf("get DC%d client: %w", dcID, err)
		}

		result, err = dcClient.API().StatsLoadAsyncGraph(ctx, &tg.StatsLoadAsyncGraphRequest{
			Token: token,
		})
	} else {
		result, err = c.api.StatsLoadAsyncGraph(ctx, &tg.StatsLoadAsyncGraphRequest{
			Token: token,
		})
	}

	if err != nil {
		return "", err
	}

	if statsGraph, ok := result.(*tg.StatsGraph); ok {
		return statsGraph.JSON.Data, nil
	}

	if statsGraphError, ok := result.(*tg.StatsGraphError); ok {
		return "", fmt.Errorf("telegram stats error: %s", statsGraphError.Error)
	}

	return "", fmt.Errorf("unexpected response type: %T", result)
}

// GetChannelAdmins fetches channel admin list.
func (c *Client) GetChannelAdmins(ctx context.Context, channel *tg.InputChannel) ([]tg.ChannelParticipantClass, error) {
	participants, err := c.api.ChannelsGetParticipants(ctx, &tg.ChannelsGetParticipantsRequest{
		Channel: channel,
		Filter:  &tg.ChannelParticipantsAdmins{},
		Offset:  0,
		Limit:   100,
		Hash:    0,
	})
	if err != nil {
		return nil, err
	}

	if cp, ok := participants.(*tg.ChannelsChannelParticipants); ok {
		return cp.Participants, nil
	}

	return nil, fmt.Errorf("unexpected response type: %T", participants)
}

// ExtractAdminInfo extracts admin info from participants.
func ExtractAdminInfo(participants []tg.ChannelParticipantClass) []AdminInfo {
	var admins []AdminInfo

	for _, p := range participants {
		switch admin := p.(type) {
		case *tg.ChannelParticipantCreator:
			admins = append(admins, AdminInfo{
				UserID:    admin.UserID,
				IsCreator: true,
				CanPost:   true,
				CanEdit:   true,
				CanDelete: true,
			})

		case *tg.ChannelParticipantAdmin:
			rights := admin.AdminRights
			admins = append(admins, AdminInfo{
				UserID:    admin.UserID,
				IsCreator: false,
				CanPost:   rights.PostMessages,
				CanEdit:   rights.EditMessages,
				CanDelete: rights.DeleteMessages,
			})
		}
	}

	return admins
}
