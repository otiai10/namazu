package p2pquake

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/otiai10/namazu/backend/internal/source"
)

const (
	// reconnectInterval is how often to reconnect to avoid 10-minute forced disconnect
	reconnectInterval = 9 * time.Minute

	// maxRetries is maximum number of connection retry attempts
	maxRetries = 10

	// initialRetryDelay is the starting delay for exponential backoff
	initialRetryDelay = 1 * time.Second

	// maxRetryDelay is the maximum delay between retries
	maxRetryDelay = 60 * time.Second
)

// Client is a WebSocket client for P2P地震情報 API
type Client struct {
	endpoint    string
	conn        *websocket.Conn
	events      chan source.Event
	done        chan struct{}
	mu          sync.Mutex
	seenIDs     map[string]struct{}
	seenIDsList []string // for LRU eviction
	maxSeenIDs  int
}

// NewClient creates a new P2P地震情報 client
func NewClient(endpoint string) *Client {
	return &Client{
		endpoint:    endpoint,
		events:      make(chan source.Event, 100),
		done:        make(chan struct{}),
		seenIDs:     make(map[string]struct{}),
		seenIDsList: make([]string, 0),
		maxSeenIDs:  1000,
	}
}

// Connect starts the WebSocket connection with automatic reconnection
func (c *Client) Connect(ctx context.Context) error {
	// Initial connection
	if err := c.connect(ctx); err != nil {
		log.Printf("Failed to establish initial connection: %v", err)
		return err
	}

	// Start read loop
	go c.readLoop(ctx)

	// Start reconnection scheduler
	go c.scheduleReconnect(ctx)

	return nil
}

// Events returns the channel for receiving events
func (c *Client) Events() <-chan source.Event {
	return c.events
}

// Close closes the connection
func (c *Client) Close() error {
	close(c.done)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// connect establishes WebSocket connection with retry
func (c *Client) connect(ctx context.Context) error {
	var lastErr error
	delay := initialRetryDelay

	for attempt := 0; attempt < maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.done:
			return fmt.Errorf("client closed")
		default:
		}

		c.mu.Lock()
		conn, _, err := websocket.DefaultDialer.DialContext(ctx, c.endpoint, nil)
		if err == nil {
			c.conn = conn
			c.mu.Unlock()
			log.Printf("Successfully connected to %s", c.endpoint)
			return nil
		}
		c.mu.Unlock()

		lastErr = err
		log.Printf("Connection attempt %d/%d failed: %v", attempt+1, maxRetries, err)

		if attempt < maxRetries-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-c.done:
				return fmt.Errorf("client closed")
			case <-time.After(delay):
				// Exponential backoff with cap
				delay *= 2
				if delay > maxRetryDelay {
					delay = maxRetryDelay
				}
			}
		}
	}

	return fmt.Errorf("failed to connect after %d attempts: %w", maxRetries, lastErr)
}

// readLoop reads messages from WebSocket
func (c *Client) readLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Read loop stopped: context cancelled")
			return
		case <-c.done:
			log.Println("Read loop stopped: client closed")
			return
		default:
		}

		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()

		if conn == nil {
			log.Println("No connection available, waiting...")
			time.Sleep(1 * time.Second)
			continue
		}

		// Read message
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Read error: %v", err)
			// Try to reconnect
			c.mu.Lock()
			c.conn = nil
			c.mu.Unlock()

			if reconnectErr := c.connect(ctx); reconnectErr != nil {
				log.Printf("Reconnection failed: %v", reconnectErr)
				time.Sleep(1 * time.Second)
			}
			continue
		}

		if messageType != websocket.TextMessage {
			continue
		}

		// Parse JSON to check code
		var rawMessage struct {
			ID   string `json:"_id"`
			Code int    `json:"code"`
		}

		if err := json.Unmarshal(data, &rawMessage); err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		// Filter for code 551 (JMAQuake) only
		if rawMessage.Code != 551 {
			continue
		}

		// Check for duplicate
		if c.isDuplicate(rawMessage.ID) {
			continue
		}

		// Parse full JMAQuake message
		var quake JMAQuake
		if err := json.Unmarshal(data, &quake); err != nil {
			log.Printf("Failed to parse JMAQuake message: %v", err)
			continue
		}

		// Add metadata
		quake.ReceivedAt = time.Now()
		quake.RawJSON = string(data)

		// Send to events channel (non-blocking)
		select {
		case c.events <- &quake:
		default:
			log.Println("Events channel full, dropping message")
		}
	}
}

// isDuplicate checks if message ID was already seen
func (c *Client) isDuplicate(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if already seen
	if _, exists := c.seenIDs[id]; exists {
		return true
	}

	// Add to seen map
	c.seenIDs[id] = struct{}{}
	c.seenIDsList = append(c.seenIDsList, id)

	// Evict oldest if over limit
	if len(c.seenIDsList) > c.maxSeenIDs {
		// Remove oldest from map
		oldestID := c.seenIDsList[0]
		delete(c.seenIDs, oldestID)

		// Remove oldest from list
		c.seenIDsList = c.seenIDsList[1:]
	}

	return false
}

// scheduleReconnect schedules automatic reconnection every 9 minutes
func (c *Client) scheduleReconnect(ctx context.Context) {
	ticker := time.NewTicker(reconnectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Reconnect scheduler stopped: context cancelled")
			return
		case <-c.done:
			log.Println("Reconnect scheduler stopped: client closed")
			return
		case <-ticker.C:
			log.Println("Scheduled reconnection (9-minute interval)")

			// Close existing connection
			c.mu.Lock()
			if c.conn != nil {
				c.conn.Close()
				c.conn = nil
			}
			c.mu.Unlock()

			// Establish new connection
			if err := c.connect(ctx); err != nil {
				log.Printf("Scheduled reconnection failed: %v", err)
			}
		}
	}
}
