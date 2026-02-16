package common

import "time"

type Graph struct {
	Data string `json:"data,omitempty"`
}

type ChannelGraphs struct {
	Mute                    Graph `json:"mute,omitempty"`
	Growth                  Graph `json:"growth,omitempty"`
	Followers               Graph `json:"followers,omitempty"`
	TopHours                Graph `json:"top_hours,omitempty"`
	Languages               Graph `json:"languages,omitempty"`
	ViewsBySource           Graph `json:"views_by_source,omitempty"`
	Interactions            Graph `json:"interactions,omitempty"`
	IvInteractions          Graph `json:"iv_interactions,omitempty"`
	NewFollowersBySource    Graph `json:"new_followers_by_source,omitempty"`
	ReactionsByEmotion      Graph `json:"reactions_by_emotion,omitempty"`
	StoryInteractions       Graph `json:"story_interactions,omitempty"`
	StoryReactionsByEmotion Graph `json:"story_reactions_by_emotion,omitempty"`
}

type LanguageStat struct {
	Name  string  `json:"name"`
	Total int     `json:"total"`
	Ratio float64 `json:"ratio"`
}

type PostsStats struct {
	DateFrom        time.Time `json:"date_from"`
	DateTo          time.Time `json:"date_to"`
	Count           int       `json:"count"`
	AvgViews        int       `json:"avg_views"`
	MedianViews     int       `json:"median_views"`
	AvgForwards     int       `json:"avg_forwards"`
	MedianForwards  int       `json:"median_forwards"`
	AvgReactions    int       `json:"avg_reactions"`
	MedianReactions int       `json:"median_reactions"`
}

type BroadcastValue struct {
	Current  float64 `json:"current"`
	Previous float64 `json:"previous"`
}

type BroadcastPeriod struct {
	MinDate time.Time `json:"min_date"`
	MaxDate time.Time `json:"max_date"`
}

type BroadcastStats struct {
	Period               BroadcastPeriod `json:"period"`
	Followers            BroadcastValue  `json:"followers"`
	ViewsPerPost         BroadcastValue  `json:"views_per_post"`
	SharesPerPost        BroadcastValue  `json:"shares_per_post"`
	ReactionsPerPost     BroadcastValue  `json:"reactions_per_post"`
	ViewsPerStory        BroadcastValue  `json:"views_per_story"`
	SharesPerStory       BroadcastValue  `json:"shares_per_story"`
	ReactionsPerStory    BroadcastValue  `json:"reactions_per_story"`
	EnabledNotifications BroadcastValue  `json:"enabled_notifications"`
}

type ChannelStats struct {
	Graphs    ChannelGraphs   `json:"graphs"`
	Languages []LanguageStat  `json:"languages,omitempty"`
	Posts     *PostsStats     `json:"posts,omitempty"`
	Broadcast *BroadcastStats `json:"broadcast,omitempty"`
}
