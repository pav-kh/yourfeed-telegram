package handler

import (
	"context"
	"log"
	"math/rand"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"github.com/pav-kh/yourfeed-telegram/internal/metrics"
	"github.com/pav-kh/yourfeed-telegram/internal/models"
	"github.com/pav-kh/yourfeed-telegram/internal/pool"
	"github.com/pav-kh/yourfeed-telegram/internal/ratelimit"
)

const (
	maxRetries            = 3
	retryDelay            = time.Second * 2
	maxMessagesPerMinute  = 30
	messageProcessTimeout = time.Second * 30
)

// UpdateHandler manages Telegram updates processing and message forwarding
type UpdateHandler struct {
	client        *telegram.Client
	channels      map[int64]*tg.InputPeerChannel
	inputPeerUser tg.InputPeerClass
	rnd           *rand.Rand
	workerPool    *pool.WorkerPool
	objectPool    *pool.ObjectPool
	rateLimit     *ratelimit.RateLimit
	metrics       *metrics.Metrics
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewUpdateHandler creates new UpdateHandler
func NewUpdateHandler() *UpdateHandler {
	ctx, cancel := context.WithCancel(context.Background())
	h := &UpdateHandler{
		channels:   make(map[int64]*tg.InputPeerChannel),
		rnd:        rand.New(rand.NewSource(time.Now().UnixNano())),
		ctx:        ctx,
		cancel:     cancel,
		objectPool: pool.NewObjectPool(),
		rateLimit:  ratelimit.NewRateLimit(maxMessagesPerMinute, messageProcessTimeout),
		metrics:    metrics.NewMetrics(),
	}

	h.workerPool = pool.NewWorkerPool()
	h.workerPool.Start(h.processMessageTask)

	go h.reportMetrics()
	return h
}

// Handle processes incoming Telegram updates
func (h *UpdateHandler) Handle(ctx context.Context, u tg.UpdatesClass) error {
	select {
	case <-h.ctx.Done():
		return h.ctx.Err()
	default:
		switch updates := u.(type) {
		case *tg.Updates:
			for _, update := range updates.Updates {
				h.processUpdate(ctx, update)
			}
		case *tg.UpdatesCombined:
			for _, update := range updates.Updates {
				h.processUpdate(ctx, update)
			}
		case *tg.UpdateShort:
			h.processUpdate(ctx, updates.Update)
		default:
			log.Printf("Unhandled update type: %T", updates)
		}
		return nil
	}
}

// processUpdate handles a single Telegram update
func (h *UpdateHandler) processUpdate(_ context.Context, update tg.UpdateClass) {
	switch u := update.(type) {
	case *tg.UpdateNewChannelMessage:
		message, ok := u.Message.(*tg.Message)
		if !ok || message.ReplyTo != nil && message.Post {
			return
		}

		peerChannel, ok := message.PeerID.(*tg.PeerChannel)
		if !ok {
			return
		}

		channel, exists := h.channels[peerChannel.ChannelID]
		if !exists {
			return
		}

		task := h.objectPool.GetMessageTask()
		task.Message = message
		task.PeerChannel = peerChannel
		task.Channel = channel
		task.Attempt = 0

		if !h.workerPool.Submit(task) {
			h.objectPool.PutMessageTask(task)
			h.metrics.IncrementDropped()
			log.Printf("Message queue is full, dropping message from channel %d", peerChannel.ChannelID)
		}
	}
}

// processMessageTask processes a single message task
func (h *UpdateHandler) processMessageTask(task *models.MessageTask) {
	start := time.Now()
	defer func() {
		h.metrics.RecordProcessingTime(time.Since(start))
		h.objectPool.PutMessageTask(task)
	}()

	ctx, cancel := context.WithTimeout(h.ctx, messageProcessTimeout)
	defer cancel()

	var lastErr error
	for attempt := task.Attempt; attempt < maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return
		default:
			if err := h.forwardMessage(ctx, task.Message, task.PeerChannel, task.Channel); err != nil {
				lastErr = err
				h.metrics.IncrementErrors()
				log.Printf("Attempt %d failed: %v", attempt+1, err)
				time.Sleep(retryDelay * time.Duration(attempt+1))
				continue
			}
			if err := h.markAsRead(ctx, task.Message, task.PeerChannel, task.Channel); err != nil {
				log.Printf("Failed to mark message as read: %v", err)
			}
			log.Printf("Message successfully forwarded from channel %d", task.PeerChannel.ChannelID)
			return
		}
	}
	log.Printf("Max retries exceeded: %v", lastErr)
}

// forwardMessage forwards a message to the target user with rate limiting
func (h *UpdateHandler) forwardMessage(ctx context.Context, message *tg.Message, peer *tg.PeerChannel, channel *tg.InputPeerChannel) error {
	if err := h.rateLimit.Acquire(ctx); err != nil {
		return err
	}

	_, err := h.client.API().MessagesForwardMessages(ctx, &tg.MessagesForwardMessagesRequest{
		FromPeer: &tg.InputPeerChannel{
			ChannelID:  peer.ChannelID,
			AccessHash: channel.AccessHash,
		},
		ID:       []int{message.ID},
		RandomID: []int64{h.rnd.Int63()},
		ToPeer:   h.inputPeerUser,
		Silent:   true,
	})
	return err
}

// markAsRead marks a message as read in the channel
func (h *UpdateHandler) markAsRead(ctx context.Context, message *tg.Message, peer *tg.PeerChannel, channel *tg.InputPeerChannel) error {
	_, err := h.client.API().ChannelsReadHistory(ctx, &tg.ChannelsReadHistoryRequest{
		Channel: &tg.InputChannel{
			ChannelID:  peer.ChannelID,
			AccessHash: channel.AccessHash,
		},
		MaxID: message.ID,
	})
	return err
}

// reportMetrics periodically logs performance metrics
func (h *UpdateHandler) reportMetrics() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			processed, dropped, errors, avgTime := h.metrics.GetStats()
			log.Printf("Metrics: processed=%d dropped=%d errors=%d avg_time=%v",
				processed, dropped, errors, avgTime)
		}
	}
}

// SetClient sets the Telegram client
func (h *UpdateHandler) SetClient(client *telegram.Client) {
	h.client = client
}

// SetChannels sets the channel mapping
func (h *UpdateHandler) SetChannels(channels map[int64]*tg.InputPeerChannel) {
	h.channels = channels
}

// SetInputPeerUser sets the target user for message forwarding
func (h *UpdateHandler) SetInputPeerUser(peer tg.InputPeerClass) {
	h.inputPeerUser = peer
}

// Shutdown performs a graceful shutdown of the update handler
func (h *UpdateHandler) Shutdown() {
	h.cancel()
	h.workerPool.Shutdown()
	h.rateLimit.Stop()
}
