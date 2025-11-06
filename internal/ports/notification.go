package ports

import (
	"context"
	"github.com/fixora/fixora/internal/domain"
)

// NotificationService defines the interface for notification services
type NotificationService interface {
	// NotifyTicketCreated sends notification when a ticket is created
	NotifyTicketCreated(ctx context.Context, ticket *domain.Ticket) error

	// NotifyTicketAssigned sends notification when a ticket is assigned
	NotifyTicketAssigned(ctx context.Context, ticket *domain.Ticket, assigneeID string) error

	// NotifyTicketUpdated sends notification when a ticket is updated
	NotifyTicketUpdated(ctx context.Context, ticket *domain.Ticket, updateType string) error

	// NotifyCommentAdded sends notification when a comment is added
	NotifyCommentAdded(ctx context.Context, comment *domain.Comment, ticket *domain.Ticket) error

	// NotifyTicketResolved sends notification when a ticket is resolved
	NotifyTicketResolved(ctx context.Context, ticket *domain.Ticket) error

	// NotifySLABreached sends notification when SLA is breached
	NotifySLABreached(ctx context.Context, ticket *domain.Ticket, slaType string) error

	// SendCustomNotification sends a custom notification
	SendCustomNotification(ctx context.Context, notification *Notification) error

	// ValidateRecipient checks if a recipient can receive notifications
	ValidateRecipient(ctx context.Context, recipientID string) error
}

// EventPublisher defines the interface for domain event publishing
type EventPublisher interface {
	// Publish publishes a domain event
	Publish(ctx context.Context, event Event) error

	// Subscribe subscribes to domain events
	Subscribe(eventType string, handler EventHandler) error

	// Unsubscribe unsubscribes from domain events
	Unsubscribe(eventType string, handler EventHandler) error
}

// Notification represents a notification message
type Notification struct {
	ID         string                 `json:"id"`
	Type       NotificationType       `json:"type"`
	Recipient  string                 `json:"recipient"`
	Subject    string                 `json:"subject"`
	Message    string                 `json:"message"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Priority   NotificationPriority   `json:"priority"`
	Channels   []NotificationChannel  `json:"channels"`
	CreatedAt  int64                  `json:"created_at"`
	ScheduledAt *int64                `json:"scheduled_at,omitempty"`
	Retries    int                    `json:"retries"`
	MaxRetries int                    `json:"max_retries"`
}

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypeTicketCreated   NotificationType = "ticket_created"
	NotificationTypeTicketAssigned  NotificationType = "ticket_assigned"
	NotificationTypeTicketUpdated   NotificationType = "ticket_updated"
	NotificationTypeCommentAdded    NotificationType = "comment_added"
	NotificationTypeTicketResolved  NotificationType = "ticket_resolved"
	NotificationTypeSLABreached     NotificationType = "sla_breached"
	NotificationTypeSystemMaintenance NotificationType = "system_maintenance"
	NotificationTypeCustom          NotificationType = "custom"
)

// NotificationPriority represents the priority of notification
type NotificationPriority string

const (
	NotificationPriorityLow    NotificationPriority = "low"
	NotificationPriorityMedium NotificationPriority = "medium"
	NotificationPriorityHigh   NotificationPriority = "high"
	NotificationPriorityCritical NotificationPriority = "critical"
)

// NotificationChannel represents the delivery channel
type NotificationChannel string

const (
	NotificationChannelEmail   NotificationChannel = "email"
	NotificationChannelSlack   NotificationChannel = "slack"
	NotificationChannelWebhook NotificationChannel = "webhook"
	NotificationChannelPush    NotificationChannel = "push"
	NotificationChannelSMS     NotificationChannel = "sms"
)

// Event represents a domain event
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Aggregate string                 `json:"aggregate"`
	AggregateID string               `json:"aggregate_id"`
	Data      map[string]interface{} `json:"data"`
	Version   int                    `json:"version"`
	CreatedAt int64                  `json:"created_at"`
}

// EventHandler handles domain events
type EventHandler interface {
	Handle(ctx context.Context, event Event) error
	EventType() string
}

// Event Types
const (
	EventTypeTicketCreated   = "ticket_created"
	EventTypeTicketAssigned  = "ticket_assigned"
	EventTypeTicketUpdated   = "ticket_updated"
	EventTypeTicketResolved  = "ticket_resolved"
	EventTypeCommentAdded    = "comment_added"
	EventTypeKBEntryCreated  = "kb_entry_created"
	EventTypeKBEntryUpdated  = "kb_entry_updated"
	EventTypeKBEntryPublished = "kb_entry_published"
)

// NewNotification creates a new notification
func NewNotification(ntype NotificationType, recipient, subject, message string, priority NotificationPriority, channels []NotificationChannel) *Notification {
	return &Notification{
		ID:         generateNotificationID(),
		Type:       ntype,
		Recipient:  recipient,
		Subject:    subject,
		Message:    message,
		Data:       make(map[string]interface{}),
		Priority:   priority,
		Channels:   channels,
		CreatedAt:  currentTimestamp(),
		MaxRetries: 3,
	}
}

// AddData adds additional data to the notification
func (n *Notification) AddData(key string, value interface{}) {
	if n.Data == nil {
		n.Data = make(map[string]interface{})
	}
	n.Data[key] = value
}

// Schedule schedules the notification for later delivery
func (n *Notification) Schedule(at int64) {
	n.ScheduledAt = &at
}

// NewEvent creates a new domain event
func NewEvent(eventType, aggregate, aggregateID string, data map[string]interface{}, version int) *Event {
	return &Event{
		ID:         generateEventID(),
		Type:       eventType,
		Aggregate:  aggregate,
		AggregateID: aggregateID,
		Data:       data,
		Version:    version,
		CreatedAt:  currentTimestamp(),
	}
}

// Notification Configuration
type NotificationConfig struct {
	EmailConfig     EmailConfig     `json:"email_config"`
	SlackConfig     SlackConfig     `json:"slack_config"`
	WebhookConfig   WebhookConfig   `json:"webhook_config"`
	DefaultChannels []NotificationChannel `json:"default_channels"`
	EnabledTypes    []NotificationType   `json:"enabled_types"`
	EnableQueue     bool               `json:"enable_queue"`
	QueueSize       int                `json:"queue_size"`
	BatchSize       int                `json:"batch_size"`
	BatchTimeoutMs  int                `json:"batch_timeout_ms"`
}

// EmailConfig represents email notification configuration
type EmailConfig struct {
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	FromEmail    string `json:"from_email"`
	FromName     string `json:"from_name"`
	UseTLS       bool   `json:"use_tls"`
	UseHTML      bool   `json:"use_html"`
}

// SlackConfig represents Slack notification configuration
type SlackConfig struct {
	WebhookURL string `json:"webhook_url"`
	Channel    string `json:"channel"`
	Username   string `json:"username"`
	IconEmoji  string `json:"icon_emoji"`
}

// WebhookConfig represents webhook notification configuration
type WebhookConfig struct {
	URL      string            `json:"url"`
	Headers  map[string]string `json:"headers"`
	Timeout  int               `json:"timeout_ms"`
	Retries  int               `json:"retries"`
}

// Default notification configuration
func DefaultNotificationConfig() NotificationConfig {
	return NotificationConfig{
		DefaultChannels: []NotificationChannel{NotificationChannelEmail, NotificationChannelSlack},
		EnabledTypes:    []NotificationType{},
		EnableQueue:     true,
		QueueSize:       1000,
		BatchSize:       10,
		BatchTimeoutMs:  5000,
	}
}

// Helper functions
func generateNotificationID() string {
	return "notif_" + timestampString()
}

func generateEventID() string {
	return "event_" + timestampString()
}

func currentTimestamp() int64 {
	return 0 // Implement with time.Now().Unix()
}

func timestampString() string {
	return "1234567890" // Implement with time.Now().Format()
}

// Notification errors
const (
	ErrNotificationNotFound    = "notification not found"
	ErrInvalidRecipient       = "invalid recipient"
	ErrChannelUnavailable     = "notification channel unavailable"
	ErrNotificationRateLimit  = "notification rate limit exceeded"
	ErrNotificationFailed     = "notification delivery failed"
	ErrEventHandlingFailed    = "event handling failed"
)