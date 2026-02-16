package stats_bot

import (
	"fmt"

	"github.com/gotd/td/tg"
	"github.com/wickedv43/TAM-backend/internal/common"
)

// extractBasicInfo populates resp with basic channel info from fullChannel.
func (s *StatsBot) extractBasicInfo(fullChannel *tg.MessagesChatFull, resp *common.AddChannelResponse) (*tg.Channel, *tg.ChannelFull) {
	var channelObj *tg.Channel
	var channelFullObj *tg.ChannelFull

	if channelFull, ok := fullChannel.FullChat.(*tg.ChannelFull); ok {
		channelFullObj = channelFull
		resp.About = channelFull.About
		resp.Subscribers = channelFull.ParticipantsCount

		if channelFull.StatsDC != 0 {
			s.log.Infof("Channel stats available on DC%d", channelFull.StatsDC)
		}

		if photo, ok := channelFull.ChatPhoto.(*tg.Photo); ok {
			resp.PhotoHash = fmt.Sprintf("photo_%d", photo.ID)
		}
	}

	for _, chat := range fullChannel.Chats {
		if channel, ok := chat.(*tg.Channel); ok {
			resp.TgID = channel.ID
			resp.Username = channel.Username
			resp.Title = channel.Title
			channelObj = channel
			break
		}
	}

	return channelObj, channelFullObj
}
