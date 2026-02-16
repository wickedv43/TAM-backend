package channels

import (
	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/wickedv43/TAM-backend/internal/common"
)

type Channels struct {
	// Dialog
	Dialog chan common.DealMessageBotRequest

	//Notifications from server
	Notification chan common.Notification

	// Show deal data
	// todo: [POST-MVP] chan interface{} -> parse like .(type)?
	Deal chan common.ShowDealDataBotRequest

	// Post
	Post chan common.SendDealDataBotRequest

	AdminMessage chan string
	CodeMessage  chan string

	AutoPost chan uuid.UUID
}

func NewChannels(_ do.Injector) (*Channels, error) {
	return &Channels{
		Dialog:       make(chan common.DealMessageBotRequest),
		Post:         make(chan common.SendDealDataBotRequest),
		AdminMessage: make(chan string, 100),
		CodeMessage:  make(chan string, 100),
		AutoPost:     make(chan uuid.UUID, 100),
		Notification: make(chan common.Notification, 100),
		Deal:         make(chan common.ShowDealDataBotRequest, 100),
	}, nil
}
