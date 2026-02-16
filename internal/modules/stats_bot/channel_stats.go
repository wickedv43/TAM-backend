package stats_bot

import (
	"context"
	"fmt"
	"time"

	"github.com/wickedv43/TAM-backend/internal/common"
)

// GetChannel resolves a channel by username, validates access, and returns stats.
func (s *StatsBot) GetChannel(req common.StatsBotAddChannelRequest) (common.AddChannelResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var resp common.AddChannelResponse

	inputChannel, err := s.tgClient.ResolveChannelUsername(ctx, req.ChannelUsername)
	if err != nil {
		return resp, fmt.Errorf("resolve channel: %w", common.ErrChannelNotFound)
	}

	fullChannel, err := s.tgClient.GetFullChannel(ctx, inputChannel)
	if err != nil {
		return resp, fmt.Errorf("get full channel: %w", common.ErrChannelNotFound)
	}

	userRole, err := s.validateChannelAccess(ctx, inputChannel, req.UserTgID)
	if err != nil {
		return resp, fmt.Errorf("validate access: %w", err)
	}

	resp.UserRole = userRole

	channel, channelFull := s.extractBasicInfo(fullChannel, &resp)

	if err = s.validateChannelRequirements(channel, resp.Subscribers); err != nil {
		return resp, fmt.Errorf("validate channel: %w", err)
	}

	if err = s.getFirstPostInfo(ctx, inputChannel, &resp); err != nil {
		s.log.Warnf("Failed to get first post info for channel %s: %v", req.ChannelUsername, err)
	}

	if err = s.getBoostsInfo(ctx, inputChannel, &resp); err != nil {
	}

	if err = s.getRecentPostsStats(ctx, inputChannel, &resp); err != nil {
	}

	resp.StatsAvailable = false
	if channelFull != nil && channelFull.CanViewStats {
		if err = s.getBroadcastStats(ctx, inputChannel, &resp); err != nil {
			s.log.Warnf("Failed to get broadcast stats for channel %s: %v", req.ChannelUsername, err)
		} else {
			resp.StatsAvailable = true
		}
	}

	return resp, nil
}
