package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gotd/td/telegram"
	"github.com/pav-kh/yourfeed-telegram/internal/config"
	"github.com/pav-kh/yourfeed-telegram/internal/handler"
	tgclient "github.com/pav-kh/yourfeed-telegram/pkg/telegram"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run initializes and runs the Telegram client with proper shutdown handling
func run() error {
	cfg, err := config.NewConfigFromEnv()
	if err != nil {
		return err
	}

	updateHandler := handler.NewUpdateHandler()
	client := tgclient.NewClient(cfg, telegram.Options{
		UpdateHandler:  updateHandler,
		SessionStorage: &telegram.FileSessionStorage{Path: "session.json"},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal, starting graceful shutdown...")
		cancel()
	}()

	if err := client.Run(ctx, func(ctx context.Context) error {
		return runBot(ctx, client, updateHandler)
	}); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	updateHandler.Shutdown()
	log.Println("Graceful shutdown completed")
	return nil
}

// runBot performs the main bot initialization and setup
func runBot(ctx context.Context, client *tgclient.Client, updateHandler *handler.UpdateHandler) error {
	// Authenticate if necessary
	if err := client.AuthenticateIfNeeded(ctx); err != nil {
		return err
	}

	// Get channels
	channels, err := client.GetChannels(ctx)
	if err != nil {
		return err
	}

	// Get recipient user
	recipientUser, err := client.GetRecipientUser(ctx)
	if err != nil {
		return err
	}

	// Initialize handler
	updateHandler.SetClient(client.Client)
	updateHandler.SetChannels(channels)
	updateHandler.SetInputPeerUser(recipientUser)

	log.Println("Bot is running and waiting for new messages...")
	<-ctx.Done()
	return nil
}
