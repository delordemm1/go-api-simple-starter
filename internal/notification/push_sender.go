package notification

import (
	"context"
	"log/slog"
)

// pushSender is a no-op implementation of the smsSender interface.
type pushSender struct {
	log *slog.Logger
}

// NewPushSender creates a new dummy SMS sender.
func NewPushSender(log *slog.Logger) smsSender {
	return &pushSender{log: log}
}

func (s *pushSender) Send(ctx context.Context, to, message string) error {
	s.log.Info("DUMMY SEND: SMS would be sent", "to", to, "message", message)
	return nil // Always succeed
}
