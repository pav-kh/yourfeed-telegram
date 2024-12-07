package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"github.com/joho/godotenv"
)

const (
	messageBufferSize = 100
	numWorkers        = 5
	maxRetries        = 3
	retryDelay        = time.Second * 2

	// Rate limiting and timeout settings
	maxMessagesPerMinute  = 30
	maxConcurrentForwards = 3
	messageProcessTimeout = time.Second * 30
)

// MessageTask represents a message processing task with all required data
type MessageTask struct {
	Message     *tg.Message
	PeerChannel *tg.PeerChannel
	Channel     *tg.InputPeerChannel
	Attempt     int
}

// WorkerPool manages a pool of worker goroutines for concurrent message processing
type WorkerPool struct {
	tasks    chan *MessageTask
	wg       sync.WaitGroup
	shutdown chan struct{}
}

// Config holds application configuration and credentials
type Config struct {
	APIID             int    // Telegram API ID
	APIHash           string // Telegram API Hash
	Phone             string // Phone number for authentication
	RecipientUsername string // Username to forward messages to
}

// NewConfigFromEnv creates Config from environment variables
func NewConfigFromEnv() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("loading .env: %w", err)
	}

	apiID, err := strconv.Atoi(os.Getenv("API_ID"))
	if err != nil {
		return nil, fmt.Errorf("parsing API_ID: %w", err)
	}

	return &Config{
		APIID:             apiID,
		APIHash:           os.Getenv("API_HASH"),
		Phone:             os.Getenv("PHONE"),
		RecipientUsername: os.Getenv("RECIPIENT_USERNAME"),
	}, nil
}

// ObjectPool manages reusable objects to reduce GC pressure
type ObjectPool struct {
	messagePool sync.Pool // Pool for MessageTask objects
	bufferPool  sync.Pool // Pool for byte buffers
}

func newObjectPool() *ObjectPool {
	return &ObjectPool{
		messagePool: sync.Pool{
			New: func() interface{} {
				return &MessageTask{}
			},
		},
		bufferPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 1024)
			},
		},
	}
}

// RateLimit implements token bucket algorithm for rate limiting
type RateLimit struct {
	ticker  *time.Ticker
	tokens  chan struct{}
	timeout time.Duration
}

func newRateLimit(rate int, timeout time.Duration) *RateLimit {
	rl := &RateLimit{
		ticker:  time.NewTicker(time.Minute / time.Duration(rate)),
		tokens:  make(chan struct{}, rate),
		timeout: timeout,
	}

	// Initialize token bucket
	for i := 0; i < rate; i++ {
		rl.tokens <- struct{}{}
	}

	go rl.refillTokens()
	return rl
}

// refillTokens periodically adds new tokens to the bucket
func (rl *RateLimit) refillTokens() {
	for range rl.ticker.C {
		select {
		case rl.tokens <- struct{}{}:
		default:
		}
	}
}

// acquire attempts to acquire a token with timeout
func (rl *RateLimit) acquire(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-rl.tokens:
		return nil
	case <-time.After(rl.timeout):
		return fmt.Errorf("rate limit timeout")
	}
}

// Metrics tracks application performance and error statistics
type Metrics struct {
	processedMessages   int64         // Total number of processed messages
	droppedMessages     int64         // Number of messages dropped due to queue overflow
	processingErrors    int64         // Number of processing errors
	avgProcessingTime   time.Duration // Average message processing time
	totalProcessingTime time.Duration // Total processing time for all messages
	mu                  sync.RWMutex  // Mutex for thread-safe metrics updates
}

// recordProcessingTime updates message processing time metrics
func (m *Metrics) recordProcessingTime(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalProcessingTime += d
	m.processedMessages++
	m.avgProcessingTime = time.Duration(int64(m.totalProcessingTime) / m.processedMessages)
}

// incrementDropped atomically increments the dropped messages counter
func (m *Metrics) incrementDropped() {
	atomic.AddInt64(&m.droppedMessages, 1)
}

// incrementErrors atomically increments the processing errors counter
func (m *Metrics) incrementErrors() {
	atomic.AddInt64(&m.processingErrors, 1)
}

// updateHandler manages Telegram updates processing and message forwarding
type updateHandler struct {
	client        *telegram.Client               // Telegram client instance
	channels      map[int64]*tg.InputPeerChannel // Mapping of channel IDs to their input peers
	inputPeerUser tg.InputPeerClass              // Target user to forward messages to
	rnd           *rand.Rand                     // Random number generator for message IDs
	workerPool    *WorkerPool                    // Pool of worker goroutines
	objectPool    *ObjectPool                    // Pool of reusable objects
	rateLimit     *RateLimit                     // Rate limiter for API calls
	metrics       *Metrics                       // Performance metrics
	ctx           context.Context                // Context for cancellation
	cancel        context.CancelFunc             // Function to cancel the context
}

// NewUpdateHandler creates new UpdateHandler
func NewUpdateHandler() *updateHandler {
	ctx, cancel := context.WithCancel(context.Background())
	h := &updateHandler{
		channels:   make(map[int64]*tg.InputPeerChannel),
		rnd:        rand.New(rand.NewSource(time.Now().UnixNano())),
		ctx:        ctx,
		cancel:     cancel,
		objectPool: newObjectPool(),
		rateLimit:  newRateLimit(maxMessagesPerMinute, messageProcessTimeout),
		metrics:    &Metrics{},
	}
	h.workerPool = newWorkerPool(h)

	// Запускаем периодический вывод метрик
	go h.reportMetrics()
	return h
}

func newWorkerPool(h *updateHandler) *WorkerPool {
	wp := &WorkerPool{
		tasks:    make(chan *MessageTask, messageBufferSize),
		shutdown: make(chan struct{}),
	}

	wp.wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go h.worker(wp)
	}

	return wp
}

// worker processes messages from the task queue
func (h *updateHandler) worker(wp *WorkerPool) {
	defer wp.wg.Done()

	for {
		select {
		case task := <-wp.tasks:
			if err := h.processMessageWithRetry(h.ctx, task); err != nil {
				log.Printf("Failed to process message after %d attempts: %v", maxRetries, err)
			}
			// Return MessageTask to the pool
			h.objectPool.messagePool.Put(task)
		case <-wp.shutdown:
			return
		}
	}
}

// processMessageWithRetry attempts to process a message with retries on failure
func (h *updateHandler) processMessageWithRetry(ctx context.Context, task *MessageTask) error {
	start := time.Now()
	defer func() {
		h.metrics.recordProcessingTime(time.Since(start))
	}()

	var lastErr error
	for attempt := task.Attempt; attempt < maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := h.forwardMessage(ctx, task.Message, task.PeerChannel, task.Channel); err != nil {
				lastErr = err
				h.metrics.incrementErrors()
				log.Printf("Attempt %d failed: %v", attempt+1, err)
				time.Sleep(retryDelay * time.Duration(attempt+1))
				continue
			}
			if err := h.markAsRead(ctx, task.Message, task.PeerChannel, task.Channel); err != nil {
				log.Printf("Failed to mark message as read: %v", err)
			}
			return nil
		}
	}
	return fmt.Errorf("max retries exceeded: %v", lastErr)
}

// Shutdown performs a graceful shutdown of the update handler
func (h *updateHandler) Shutdown() {
	h.cancel()
	close(h.workerPool.shutdown)
	h.workerPool.wg.Wait()
}

// Handle processes incoming Telegram updates
func (h *updateHandler) Handle(ctx context.Context, u tg.UpdatesClass) error {
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
func (h *updateHandler) processUpdate(ctx context.Context, update tg.UpdateClass) {
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

		// Get MessageTask from the pool
		task := h.objectPool.messagePool.Get().(*MessageTask)
		task.Message = message
		task.PeerChannel = peerChannel
		task.Channel = channel
		task.Attempt = 0

		select {
		case <-ctx.Done():
			h.objectPool.messagePool.Put(task)
			return
		case h.workerPool.tasks <- task:
			log.Printf("Queued message from channel %d for processing", peerChannel.ChannelID)
		default:
			h.objectPool.messagePool.Put(task)
			h.metrics.incrementDropped()
			log.Printf("Message queue is full, dropping message from channel %d", peerChannel.ChannelID)
		}
	}
}

// forwardMessage forwards a message to the target user with rate limiting
func (h *updateHandler) forwardMessage(ctx context.Context, message *tg.Message, peer *tg.PeerChannel, channel *tg.InputPeerChannel) error {
	if err := h.rateLimit.acquire(ctx); err != nil {
		return fmt.Errorf("rate limit: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, messageProcessTimeout)
	defer cancel()

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
func (h *updateHandler) markAsRead(ctx context.Context, message *tg.Message, peer *tg.PeerChannel, channel *tg.InputPeerChannel) error {
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
func (h *updateHandler) reportMetrics() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.metrics.mu.RLock()
			log.Printf("Performance metrics: processed=%d dropped=%d errors=%d avg_time=%v",
				h.metrics.processedMessages,
				h.metrics.droppedMessages,
				h.metrics.processingErrors,
				h.metrics.avgProcessingTime)
			h.metrics.mu.RUnlock()
		}
	}
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run initializes and runs the Telegram client with proper shutdown handling
func run() error {
	cfg, err := NewConfigFromEnv()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	handler := NewUpdateHandler()
	client := telegram.NewClient(cfg.APIID, cfg.APIHash, telegram.Options{
		UpdateHandler:  handler,
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
		return runBot(ctx, client, handler, cfg)
	}); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("client run error: %w", err)
	}

	handler.Shutdown()
	log.Println("Graceful shutdown completed")
	return nil
}

// runBot performs the main bot initialization and setup
func runBot(ctx context.Context, client *telegram.Client, handler *updateHandler, cfg *Config) error {
	// Setup authentication code prompt
	codePrompt := func(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
		var code string
		fmt.Print("Enter code: ")
		_, err := fmt.Scanln(&code)
		return code, err
	}

	// Perform authentication if necessary
	flow := auth.NewFlow(
		auth.CodeOnly(cfg.Phone, auth.CodeAuthenticatorFunc(codePrompt)),
		auth.SendCodeOptions{},
	)
	if err := client.Auth().IfNecessary(ctx, flow); err != nil {
		if rpcErr, ok := err.(*telegram.Error); ok && rpcErr.Code == 420 {
			waitTime := rpcErr.Argument
			log.Printf("FLOOD_WAIT: need to wait %d seconds before retrying", waitTime)
			time.Sleep(time.Duration(waitTime) * time.Second)
			return client.Auth().IfNecessary(ctx, flow)
		}
		return fmt.Errorf("authentication error: %w", err)
	}

	// Initialize channels and user for message forwarding
	if err := initializeChannelsAndUser(ctx, client, handler, cfg); err != nil {
		return err
	}

	log.Println("Bot is running and waiting for new messages...")
	<-ctx.Done()
	return nil
}

// initializeChannelsAndUser sets up channel list and target user for message forwarding
func initializeChannelsAndUser(ctx context.Context, client *telegram.Client, handler *updateHandler, cfg *Config) error {
	// Get channel list
	dialogs, err := client.API().MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		Limit:      100,
		OffsetDate: 0,
		OffsetID:   0,
		OffsetPeer: &tg.InputPeerEmpty{},
	})
	if err != nil {
		return fmt.Errorf("error getting dialogs: %v", err)
	}

	var chats []tg.ChatClass
	switch d := dialogs.(type) {
	case *tg.MessagesDialogs:
		chats = d.Chats
	case *tg.MessagesDialogsSlice:
		chats = d.Chats
	default:
		return fmt.Errorf("invalid dialog data type: %T", dialogs)
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
		return fmt.Errorf("no subscribed channels")
	}

	// Get user for message forwarding
	userRes, err := client.API().ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: cfg.RecipientUsername,
	})
	if err != nil {
		return fmt.Errorf("error getting user: %v", err)
	}

	var inputPeerUser tg.InputPeerClass
	for _, user := range userRes.Users {
		if usr, ok := user.(*tg.User); ok {
			inputPeerUser = &tg.InputPeerUser{
				UserID:     usr.ID,
				AccessHash: usr.AccessHash,
			}
			break
		}
	}

	if inputPeerUser == nil {
		return fmt.Errorf("user not found")
	}

	handler.client = client
	handler.channels = channels
	handler.inputPeerUser = inputPeerUser

	return nil
}
