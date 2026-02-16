package tg_api

import (
	"context"

	mtStorage "github.com/gotd/contrib/storage"
	"github.com/gotd/td/telegram/query"
	"github.com/gotd/td/telegram/updates"
	"github.com/gotd/td/tgerr"
	"github.com/pkg/errors"
)

// Run starts the Telegram client and optional peer storage collection.
func (t *Client) Run(ctx context.Context, FillPeerStorage bool) error {
	return t.waiter.Run(ctx, func(ctx context.Context) error {
		if err := t.client.Run(ctx, func(ctx context.Context) error {
			phone, _ := t.Flow.Auth.Phone(ctx)
			t.log.Info("Start register agent with phone: ", phone)
			if err := t.client.Auth().IfNecessary(ctx, *t.Flow); err != nil {
				if rpcErr, ok := tgerr.As(err); ok {
					if waitDuration, ok := tgerr.AsFloodWait(err); ok {
						t.log.Errorf("Telegram RPC Error: %s (Code: %d) - wait %v before retry", rpcErr.Type, rpcErr.Code, waitDuration)
					} else {
						t.log.Errorf("Telegram RPC Error: %s (Code: %d)", rpcErr.Type, rpcErr.Code)
					}
				} else {
					t.log.Errorf("Auth error (not RPC): %v", err)
				}

				return errors.Wrap(err, "auth")
			}

			self, err := t.client.Self(ctx)
			if err != nil {
				return errors.Wrap(err, "call self")
			}

			t.cfg.UserBot.UserID = self.ID
			t.cfg.UserBot.Username = "@" + self.Username
			t.log.Debugf("StatsBot: [%d] %s", t.cfg.UserBot.UserID, t.cfg.UserBot.Username)

			if FillPeerStorage {
				t.log.Info("Filling peer storage from dialogs to cache entities")
				collector := mtStorage.CollectPeers(t.peerDB)
				if err = collector.Dialogs(ctx, query.GetDialogs(t.api).Iter()); err != nil {
					return errors.Wrap(err, "collect peers")
				}
				t.log.Info("Filled")
			}

			return t.UpdatesRecovery.Run(ctx, t.api, self.ID, updates.AuthOptions{
				IsBot: self.Bot,
				OnStart: func(ctx context.Context) {
					t.log.Info("Update recovery initialized and started, listening for events")
				},
			})
		}); err != nil {
			return errors.Wrap(err, "run")
		}
		return nil
	})
}
