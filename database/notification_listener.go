package database

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/sted/smoothdb/logging"
)

// ListenerConfig holds configuration for the notification listener
type ListenerConfig struct {
	Channel        string
	Enabled        bool
	ReconnectDelay time.Duration
}

// DefaultListenerConfig returns the default configuration
func DefaultListenerConfig() *ListenerConfig {
	return &ListenerConfig{
		Channel:        "smoothdb",
		Enabled:        true,
		ReconnectDelay: 5 * time.Second,
	}
}

// NotificationHandler defines the interface for handling notifications
type NotificationHandler func(ctx context.Context, payload string) error

// NotificationListener implements the PostgreSQL LISTEN/NOTIFY functionality
type NotificationListener struct {
	config     *ListenerConfig
	logger     *logging.Logger
	connString string
	handlers   map[string]NotificationHandler
	mu         sync.RWMutex
	running    bool
	cancel     context.CancelFunc
	stopped    chan struct{}
}

// NewNotificationListener creates a new notification listener
func NewNotificationListener(connString string, config *ListenerConfig, logger *logging.Logger) *NotificationListener {
	if config == nil {
		config = DefaultListenerConfig()
	}
	return &NotificationListener{
		config:     config,
		logger:     logger,
		connString: connString,
		handlers:   make(map[string]NotificationHandler),
	}
}

// RegisterHandler registers a handler for a specific notification command
func (l *NotificationListener) RegisterHandler(command string, handler NotificationHandler) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.handlers[command] = handler
}

// Start begins listening for notifications
func (l *NotificationListener) Start(ctx context.Context) error {
	if !l.config.Enabled {
		if l.logger != nil {
			l.logger.Info().Msg("Notification listener is disabled")
		}
		return nil
	}

	l.mu.Lock()
	if l.running {
		l.mu.Unlock()
		return fmt.Errorf("listener is already running")
	}
	l.running = true
	ctx, l.cancel = context.WithCancel(ctx)
	l.stopped = make(chan struct{})
	stopped := l.stopped
	l.mu.Unlock()

	go func() {
		defer close(stopped)
		l.listenLoop(ctx)
	}()
	return nil
}

// Stop stops the listener, interrupting a pending WaitForNotification, and
// waits for the loop to exit, so its dedicated connection is closed when
// Stop returns — also when racing another Stop. It must not be called from
// a notification handler, which runs on the loop it would wait for.
func (l *NotificationListener) Stop() {
	l.mu.Lock()
	if l.running {
		l.running = false
		l.cancel()
	}
	stopped := l.stopped
	l.mu.Unlock()
	if stopped != nil {
		<-stopped
	}
}

// listenLoop maintains the connection and processes notifications
func (l *NotificationListener) listenLoop(ctx context.Context) {
	for {
		err := l.connectAndListen(ctx)
		if ctx.Err() != nil {
			// Stopped: WaitForNotification reports the canceled
			// context as an error, not worth logging
			return
		}
		if err != nil {
			if l.logger != nil {
				l.logger.Err(err).Msg("Notification listener error")
			}
			select {
			case <-time.After(l.config.ReconnectDelay):
			case <-ctx.Done():
				return
			}
		}
	}
}

// connectAndListen establishes a connection and listens for notifications
func (l *NotificationListener) connectAndListen(ctx context.Context) error {
	conn, err := pgx.Connect(ctx, l.connString)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() {
		// Close cleanly even when ctx is already canceled (i.e. during
		// Stop), so Postgres sees a Terminate message instead of a reset
		closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		conn.Close(closeCtx)
	}()

	_, err = conn.Exec(ctx, fmt.Sprintf("LISTEN %s", l.config.Channel))
	if err != nil {
		return fmt.Errorf("failed to start listening: %w", err)
	}

	if l.logger != nil {
		l.logger.Info().Msgf("Notification listener started, channel: %s", l.config.Channel)
	}

	for {
		notification, err := conn.WaitForNotification(ctx)
		if err != nil {
			return fmt.Errorf("error waiting for notification: %w", err)
		}
		if notification != nil {
			l.handleNotification(ctx, notification.Payload)
		}
	}
}

// handleNotification processes received notifications
func (l *NotificationListener) handleNotification(ctx context.Context, payload string) {
	if l.logger != nil {
		l.logger.Debug().Msgf("Received notification, payload: %s", payload)
	}

	// Parse the payload
	// Expected formats:
	// "reload schema" - reload all databases
	// "reload schema <database>" - reload specific database

	if payload == "" {
		// Default to reload schema for backward compatibility
		payload = "reload schema"
	}

	parts := strings.Fields(payload)
	if len(parts) == 0 {
		return
	}

	command := parts[0]
	if len(parts) > 1 && parts[0] == "reload" && parts[1] == "schema" {
		command = "reload schema"
	}

	l.mu.RLock()
	handler, ok := l.handlers[command]
	l.mu.RUnlock()

	if ok {
		err := handler(ctx, payload)
		if err != nil {
			if l.logger != nil {
				l.logger.Err(err).Msgf("Failed to handle notification, command: %s", command)
			}
		}
	} else {
		if l.logger != nil {
			l.logger.Warn().Msgf("No handler registered for command %s", command)
		}
	}
}

// CreateSchemaReloadHandler creates a handler for schema reload notifications
func CreateSchemaReloadHandler(dbe *DbEngine) NotificationHandler {
	return func(ctx context.Context, payload string) error {
		parts := strings.Fields(payload)

		// Check if a specific database is mentioned
		if len(parts) > 2 {
			dbName := parts[2]
			return dbe.ReloadDatabaseSchema(ctx, dbName)
		}

		// Reload all databases
		return dbe.ReloadAllSchemas(ctx)
	}
}
