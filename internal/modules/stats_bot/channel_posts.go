package stats_bot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gotd/td/tg"
	"github.com/wickedv43/TAM-backend/internal/common"
)

// getFirstPostInfo fetches the first post date for the channel.
func (s *StatsBot) getFirstPostInfo(ctx context.Context, channel *tg.InputChannel, resp *common.AddChannelResponse) error {
	messages, totalCount, err := s.tgClient.GetChannelHistory(ctx, channel, 1, 1, -1)
	if err != nil {
		return fmt.Errorf("get first message: %w", err)
	}

	resp.TotalPosts = totalCount

	if len(messages) > 0 {
		switch msg := messages[0].(type) {
		case *tg.Message:
			resp.FirstPostDate = time.Unix(int64(msg.Date), 0)
			s.log.Debugf("FirstPostDate set from Message: %v (unix: %d)", resp.FirstPostDate, msg.Date)
		case *tg.MessageService:
			resp.FirstPostDate = time.Unix(int64(msg.Date), 0)
			s.log.Debugf("FirstPostDate set from MessageService: %v (unix: %d)", resp.FirstPostDate, msg.Date)
		default:
			s.log.Warnf("Unknown message type: %T", messages[0])
		}
	} else {
		s.log.Warnf("No messages found for channel, FirstPostDate will be zero")
	}

	return nil
}

// getRecentPostsStats fetches stats from the last 100 posts.
func (s *StatsBot) getRecentPostsStats(ctx context.Context, channel *tg.InputChannel, resp *common.AddChannelResponse) error {

	if resp.Stats == nil {
		resp.Stats = &common.ChannelStats{}
	}

	messages, _, err := s.tgClient.GetChannelHistory(ctx, channel, 100, 0, 0)
	if err != nil {
		return fmt.Errorf("get recent messages: %w", err)
	}

	var views, forwards, reactions []int
	var dates []time.Time

	for _, m := range messages {
		msg, ok := m.(*tg.Message)
		if !ok {
			continue
		}

		if msg.Views == 0 {
			continue
		}

		views = append(views, msg.Views)
		dates = append(dates, time.Unix(int64(msg.Date), 0))

		if msg.Forwards != 0 {
			forwards = append(forwards, msg.Forwards)
		} else {
			forwards = append(forwards, 0)
		}

		reactionCount := 0
		if !msg.Reactions.Zero() {
			for _, r := range msg.Reactions.Results {
				reactionCount += r.Count
			}
		}
		reactions = append(reactions, reactionCount)
	}

	resp.MedianViews = CalculateMedian(views)
	resp.MedianForwards = CalculateMedian(forwards)
	resp.MedianReactions = CalculateMedian(reactions)

	var avgViews, avgForwards, avgReactions int
	if len(views) > 0 {
		sum := 0
		for _, v := range views {
			sum += v
		}
		avgViews = sum / len(views)
		resp.AverageViews = avgViews
	}

	if len(forwards) > 0 {
		sum := 0
		for _, v := range forwards {
			sum += v
		}
		avgForwards = sum / len(forwards)
	}

	if len(reactions) > 0 {
		sum := 0
		for _, v := range reactions {
			sum += v
		}
		avgReactions = sum / len(reactions)
	}

	if len(dates) > 0 {
		resp.Stats.Posts = &common.PostsStats{
			DateFrom:        dates[len(dates)-1],
			DateTo:          dates[0],
			Count:           len(views),
			AvgViews:        avgViews,
			MedianViews:     resp.MedianViews,
			AvgForwards:     avgForwards,
			MedianForwards:  resp.MedianForwards,
			AvgReactions:    avgReactions,
			MedianReactions: resp.MedianReactions,
		}
	}

	return nil
}

// getBoostsInfo fetches channel boosts and premium subscriber info.
func (s *StatsBot) getBoostsInfo(ctx context.Context, channel *tg.InputChannel, resp *common.AddChannelResponse) error {
	boosts, err := s.tgClient.GetBoostsStatus(ctx, channel)
	if err != nil {
		return fmt.Errorf("get boosts: %w", err)
	}

	resp.BoostLevel = boosts.Level
	resp.Boosts = boosts.Boosts

	if !boosts.PremiumAudience.Zero() {
		part := float64(boosts.PremiumAudience.Part)
		total := float64(boosts.PremiumAudience.Total)
		if total > 0 {
			resp.PremiumPercent = (part / total) * 100
			resp.PremiumSubscribers = int(part)
		}
	}

	return nil
}

// getBroadcastStats fetches channel broadcast statistics and graphs.
func (s *StatsBot) getBroadcastStats(ctx context.Context, channel *tg.InputChannel, resp *common.AddChannelResponse) error {
	broadcast, actualDC, err := s.tgClient.GetBroadcastStats(ctx, channel, true)
	if err != nil {
		return fmt.Errorf("get broadcast stats: %w", err)
	}

	if actualDC > 0 {
		s.log.Infof("Stats retrieved from DC%d", actualDC)
	}

	if resp.Stats == nil {
		resp.Stats = &common.ChannelStats{}
	}

	s.loadGraph(ctx, "mute", broadcast.MuteGraph, &resp.Stats.Graphs.Mute, actualDC)
	s.loadGraph(ctx, "growth", broadcast.GrowthGraph, &resp.Stats.Graphs.Growth, actualDC)
	s.loadGraph(ctx, "followers", broadcast.FollowersGraph, &resp.Stats.Graphs.Followers, actualDC)
	s.loadGraph(ctx, "top_hours", broadcast.TopHoursGraph, &resp.Stats.Graphs.TopHours, actualDC)
	s.loadGraph(ctx, "languages", broadcast.LanguagesGraph, &resp.Stats.Graphs.Languages, actualDC)
	s.loadGraph(ctx, "views_by_source", broadcast.ViewsBySourceGraph, &resp.Stats.Graphs.ViewsBySource, actualDC)
	s.loadGraph(ctx, "interactions", broadcast.InteractionsGraph, &resp.Stats.Graphs.Interactions, actualDC)
	s.loadGraph(ctx, "iv_interactions", broadcast.IvInteractionsGraph, &resp.Stats.Graphs.IvInteractions, actualDC)
	s.loadGraph(ctx, "new_followers_by_source", broadcast.NewFollowersBySourceGraph, &resp.Stats.Graphs.NewFollowersBySource, actualDC)
	s.loadGraph(ctx, "reactions_by_emotion", broadcast.ReactionsByEmotionGraph, &resp.Stats.Graphs.ReactionsByEmotion, actualDC)
	s.loadGraph(ctx, "story_interactions", broadcast.StoryInteractionsGraph, &resp.Stats.Graphs.StoryInteractions, actualDC)
	s.loadGraph(ctx, "story_reactions_by_emotion", broadcast.StoryReactionsByEmotionGraph, &resp.Stats.Graphs.StoryReactionsByEmotion, actualDC)

	if broadcast.EnabledNotifications.Total > 0 {
		notificationsOn := float64(broadcast.EnabledNotifications.Part) / float64(broadcast.EnabledNotifications.Total)
		resp.NotificationsOn = float64(int(notificationsOn*10000)) / 10000
	}

	if resp.Stats.Graphs.Languages.Data != "" {
		resp.Stats.Languages = s.parseLanguagesStats(resp.Stats.Graphs.Languages.Data)
	}

	resp.Subscribers = int(broadcast.Followers.Current)

	resp.Stats.Broadcast = &common.BroadcastStats{
		Period: common.BroadcastPeriod{
			MinDate: time.Unix(int64(broadcast.Period.MinDate), 0),
			MaxDate: time.Unix(int64(broadcast.Period.MaxDate), 0),
		},
		Followers: common.BroadcastValue{
			Current:  broadcast.Followers.Current,
			Previous: broadcast.Followers.Previous,
		},
		ViewsPerPost: common.BroadcastValue{
			Current:  broadcast.ViewsPerPost.Current,
			Previous: broadcast.ViewsPerPost.Previous,
		},
		SharesPerPost: common.BroadcastValue{
			Current:  broadcast.SharesPerPost.Current,
			Previous: broadcast.SharesPerPost.Previous,
		},
		ReactionsPerPost: common.BroadcastValue{
			Current:  broadcast.ReactionsPerPost.Current,
			Previous: broadcast.ReactionsPerPost.Previous,
		},
		ViewsPerStory: common.BroadcastValue{
			Current:  broadcast.ViewsPerStory.Current,
			Previous: broadcast.ViewsPerStory.Previous,
		},
		SharesPerStory: common.BroadcastValue{
			Current:  broadcast.SharesPerStory.Current,
			Previous: broadcast.SharesPerStory.Previous,
		},
		ReactionsPerStory: common.BroadcastValue{
			Current:  broadcast.ReactionsPerStory.Current,
			Previous: broadcast.ReactionsPerStory.Previous,
		},
		EnabledNotifications: common.BroadcastValue{
			Current:  broadcast.EnabledNotifications.Part,
			Previous: broadcast.EnabledNotifications.Total,
		},
	}

	return nil
}

func (s *StatsBot) loadGraph(ctx context.Context, name string, graph tg.StatsGraphClass, target *common.Graph, dcID int) {
	switch g := graph.(type) {
	case *tg.StatsGraphAsync:
		if g.Token != "" {
			s.log.Debugf("Loading async graph: %s from DC%d", name, dcID)

			graphData, err := s.tgClient.LoadAsyncGraph(ctx, g.Token, dcID)
			if err != nil {
				s.log.Warnf("Failed to load graph %s: %v", name, err)
				return
			}

			target.Data = graphData
		}

	case *tg.StatsGraph:
		if g.JSON.Data != "" {
			s.log.Debugf("Using sync graph: %s", name)
			target.Data = g.JSON.Data
		}

	case *tg.StatsGraphError:
		s.log.Warnf("Graph %s returned error: %s", name, g.Error)

	default:
		s.log.Debugf("Unknown graph type for %s", name)
	}
}

func (s *StatsBot) parseLanguagesStats(graphJSON string) []common.LanguageStat {
	type GraphData struct {
		Columns [][]interface{}   `json:"columns"`
		Names   map[string]string `json:"names"`
	}

	var data GraphData
	if err := json.Unmarshal([]byte(graphJSON), &data); err != nil {
		s.log.Warnf("Failed to parse languages graph: %v", err)
		return nil
	}

	if len(data.Columns) <= 1 {
		return nil
	}

	var result []common.LanguageStat
	var langTotal float64

	type langData struct {
		key   string
		name  string
		total float64
	}
	var languages []langData

	for _, col := range data.Columns[1:] {
		if len(col) < 2 {
			continue
		}

		key, ok := col[0].(string)
		if !ok {
			continue
		}

		name := data.Names[key]
		if name == "" {
			name = key
		}

		var sum float64
		for _, val := range col[1:] {
			if numVal, ok := val.(float64); ok {
				sum += numVal
			}
		}

		languages = append(languages, langData{
			key:   key,
			name:  name,
			total: sum,
		})
		langTotal += sum
	}

	if langTotal > 0 {
		for _, lang := range languages {
			ratio := lang.total / langTotal
			ratio = float64(int(ratio*10000)) / 10000

			result = append(result, common.LanguageStat{
				Name:  lang.name,
				Total: int(lang.total),
				Ratio: ratio,
			})
		}
	}

	return result
}
