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
	conn       *pgx.Conn
	handlers   map[string]NotificationHandler
	mu         sync.RWMutex
	running    bool
	stop       chan struct{}
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
		stop:       make(chan struct{}),
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
		l.logger.Info().Msg("Notification listener is disabled")
		return nil
	}

	l.mu.Lock()
	if l.running {
		l.mu.Unlock()
		return fmt.Errorf("listener is already running")
	}
	l.running = true
	l.mu.Unlock()

	go l.listenLoop(ctx)
	return nil
}

// Stop stops the listener
func (l *NotificationListener) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.running {
		close(l.stop)
		l.running = false
	}
}

// listenLoop maintains the connection and processes notifications
func (l *NotificationListener) listenLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-l.stop:
			return
		default:
			err := l.connectAndListen(ctx)
			if err != nil {
				l.logger.Err(err).Msg("Notification listener error")
				select {
				case <-time.After(l.config.ReconnectDelay):
					continue
				case <-ctx.Done():
					return
				case <-l.stop:
					return
				}
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
	defer conn.Close(ctx)

	l.conn = conn

	_, err = conn.Exec(ctx, fmt.Sprintf("LISTEN %s", l.config.Channel))
	if err != nil {
		return fmt.Errorf("failed to start listening: %w", err)
	}

	l.logger.Info().Msgf("Notification listener started, channel: %s", l.config.Channel)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-l.stop:
			return nil
		default:
			notification, err := conn.WaitForNotification(ctx)
			if err != nil {
				return fmt.Errorf("error waiting for notification: %w", err)
			}

			if notification != nil {
				l.handleNotification(ctx, notification.Payload)
			}
		}
	}
}

// handleNotification processes received notifications
func (l *NotificationListener) handleNotification(ctx context.Context, payload string) {
	l.logger.Debug().Msgf("Received notification, payload: %s", payload)

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
			l.logger.Err(err).Msgf("Failed to handle notification, command: %s", command)
		}
	} else {
		l.logger.Warn().Msgf("No handler registered for command %s", command)
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
