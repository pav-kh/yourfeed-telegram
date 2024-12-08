package telegram

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"github.com/pav-kh/yourfeed-telegram/internal/config"
)

// Client wraps telegram.Client with additional functionality
type Client struct {
	*telegram.Client
	cfg *config.Config
}

// NewClient creates a new Telegram client with the given configuration
func NewClient(cfg *config.Config, opts telegram.Options) *Client {
	client := telegram.NewClient(cfg.APIID, cfg.APIHash, opts)
	return &Client{
		Client: client,
		cfg:    cfg,
	}
}

// AuthenticateIfNeeded performs authentication if necessary
func (c *Client) AuthenticateIfNeeded(ctx context.Context) error {
	// Setup authentication code prompt
	codePrompt := func(_ context.Context, sentCode *tg.AuthSentCode) (string, error) {
		var code string
		fmt.Print("Enter confirmation code: ")
		_, err := fmt.Scanln(&code)
		return code, err
	}

	flow := auth.NewFlow(
		auth.CodeOnly(c.cfg.Phone, auth.CodeAuthenticatorFunc(codePrompt)),
		auth.SendCodeOptions{},
	)

	if err := c.Auth().IfNecessary(ctx, flow); err != nil {
		if rpcErr, ok := err.(*telegram.Error); ok && rpcErr.Code == 420 {
			waitTime := rpcErr.Argument
			log.Printf("FLOOD_WAIT: need to wait %d seconds before retrying", waitTime)
			time.Sleep(time.Duration(waitTime) * time.Second)
			return c.Auth().IfNecessary(ctx, flow)
		}
		return fmt.Errorf("authentication error: %w", err)
	}

	return nil
}

// GetChannels returns a list of subscribed channels
func (c *Client) GetChannels(ctx context.Context) (map[int64]*tg.InputPeerChannel, error) {
	dialogs, err := c.API().MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		Limit:      100,
		OffsetDate: 0,
		OffsetID:   0,
		OffsetPeer: &tg.InputPeerEmpty{},
	})
	if err != nil {
		return nil, fmt.Errorf("error getting dialogs: %v", err)
	}

	var chats []tg.ChatClass
	switch d := dialogs.(type) {
	case *tg.MessagesDialogs:
		chats = d.Chats
	case *tg.MessagesDialogsSlice:
		chats = d.Chats
	default:
		return nil, fmt.Errorf("invalid dialog data type: %T", dialogs)
	}

	channels := make(map[int64]*tg.InputPeerChannel)
	for _, chat := range chats {
		if ch, ok := chat.(*tg.Channel); ok && !ch.Megagroup {
			channels[ch.ID] = &tg.InputPeerChannel{
				ChannelID:  ch.ID,
				AccessHash: ch.AccessHash,
			}
		}
	}

	if len(channels) == 0 {
		return nil, fmt.Errorf("no subscribed channels")
	}

	return channels, nil
}

// GetRecipientUser returns the target user for message forwarding
func (c *Client) GetRecipientUser(ctx context.Context) (tg.InputPeerClass, error) {
	userRes, err := c.API().ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: c.cfg.RecipientUsername,
	})
	if err != nil {
		return nil, fmt.Errorf("error getting user: %v", err)
	}

	for _, user := range userRes.Users {
		if usr, ok := user.(*tg.User); ok {
			return &tg.InputPeerUser{
				UserID:     usr.ID,
				AccessHash: usr.AccessHash,
			}, nil
		}
	}

	return nil, fmt.Errorf("user not found")
}
