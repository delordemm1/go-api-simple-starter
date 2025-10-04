package notification

import (
	"context"
	"log/slog"
)

// --- Constants for Type Safety ---
type Channel string
type Priority string

const (
	ChannelEmail Channel = "email"
	ChannelSMS   Channel = "sms"
	ChannelPush  Channel = "push"
)

const (
	PriorityHigh   Priority = "high"
	PriorityMedium Priority = "medium"
	PriorityLow    Priority = "low"
)

// --- Data Structures ---

// Content holds the specific message data for each channel.
// A notification can contain content for multiple channels simultaneously.
type Content struct {
	EmailSubject   string
	EmailHTMLBody  string
	SMSText        string
	PushTitle      string
	PushBody       string
	PushDataObject map[string]string // For custom data payloads in push notifications
}

// Notification is the universal object used to send any notification.
type Notification struct {
	Recipient string    // Can be an email address, phone number, or device token
	Channels  []Channel // A list of channels to send to
	Priority  Priority
	Content   Content
}

// --- Internal Sender Interfaces ---
// These are not exposed outside the package.
type emailSender interface {
	Send(ctx context.Context, to, subject, htmlBody string) error
}
type smsSender interface {
	Send(ctx context.Context, to, message string) error
}

// --- Public Service ---

// Service is the main interface for the notification system.
type Service interface {
	Send(ctx context.Context, n Notification) error
}

// service is the concrete implementation.
type service struct {
	log         *slog.Logger
	emailSender emailSender
	smsSender   smsSender
}

// NewService creates a new notification service.
func NewService(log *slog.Logger, emailSender emailSender, smsSender smsSender) Service {
	return &service{
		log:         log,
		emailSender: emailSender,
		smsSender:   smsSender,
	}
}

// Send acts as a dispatcher, routing the notification to the correct channel sender.
func (s *service) Send(ctx context.Context, n Notification) error {
	for _, channel := range n.Channels {
		// Launch each channel send in a separate goroutine for speed.
		go func(ch Channel) {
			var err error
			switch ch {
			case ChannelEmail:
				s.log.Info("dispatching email notification", "recipient", n.Recipient)
				err = s.emailSender.Send(ctx, n.Recipient, n.Content.EmailSubject, n.Content.EmailHTMLBody)
			case ChannelSMS:
				s.log.Info("dispatching sms notification", "recipient", n.Recipient)
				err = s.smsSender.Send(ctx, n.Recipient, n.Content.SMSText)
			case ChannelPush:
				s.log.Warn("push notifications are not yet implemented")
				// err = s.pushSender.Send(...)
			default:
				s.log.Warn("unsupported notification channel", "channel", ch)
			}

			if err != nil {
				// We can't return an error here, so we must log it for monitoring.
				s.log.Error("failed to send notification", "channel", ch, "recipient", n.Recipient, "error", err)
			}
		}(channel)
	}
	return nil // Return immediately
}
