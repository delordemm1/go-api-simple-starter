package notification

import (
	"context"
	"errors"
	"log/slog"

	"github.com/delordemm1/go-api-simple-starter/internal/notification/templates"
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
	// SendTemplateAny renders a template by ID with the provided data and dispatches across channels.
	// Prefer the typed helper SendTemplate[T](...) for compile-time safety.
	SendTemplateAny(ctx context.Context, recipient string, channels []Channel, priority Priority, templateID string, data any) error
}

// service is the concrete implementation.
type service struct {
	log              *slog.Logger
	emailSender      emailSender
	smsSender        smsSender
	templateRenderer templates.Renderer
}

// NewService creates a new notification service.
func NewService(log *slog.Logger, emailSender emailSender, smsSender smsSender, renderer templates.Renderer) Service {
	return &service{
		log:              log,
		emailSender:      emailSender,
		smsSender:        smsSender,
		templateRenderer: renderer,
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

// SendTemplateAny renders a template by ID with the provided data and dispatches across channels.
func (s *service) SendTemplateAny(ctx context.Context, recipient string, channels []Channel, priority Priority, templateID string, data any) error {
	if s.templateRenderer == nil {
		s.log.Error("template renderer is not configured")
		return errors.New("template renderer not configured")
	}
	rendered, err := s.templateRenderer.RenderAny(ctx, templateID, data)
	if err != nil {
		return err
	}

	n := Notification{
		Recipient: recipient,
		Channels:  channels,
		Priority:  priority,
		Content: Content{
			EmailSubject:  rendered.Subject,
			EmailHTMLBody: rendered.EmailHTML,
			SMSText:       rendered.SMSText,
			PushTitle:     rendered.PushTitle,
			PushBody:      rendered.PushBody,
		},
	}
	return s.Send(ctx, n)
}

// SendTemplate is a typed helper that preserves compile-time type-safety via a Handle[T].
func SendTemplate[T any](ctx context.Context, s Service, h templates.Handle[T], recipient string, channels []Channel, priority Priority, data T) error {
	return s.SendTemplateAny(ctx, recipient, channels, priority, h.ID(), data)
}
