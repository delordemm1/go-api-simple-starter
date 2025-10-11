package notification

import (
	"context"
	"log/slog"
)

// dummySMSSender is a no-op implementation of the smsSender interface.
type dummySMSSender struct {
	log *slog.Logger
}

// NewDummySMSSender creates a new dummy SMS sender.
func NewDummySMSSender(log *slog.Logger) smsSender {
	return &dummySMSSender{log: log}
}

func (s *dummySMSSender) Send(ctx context.Context, to, message string) error {
	s.log.Info("DUMMY SEND: SMS would be sent", "to", to, "message", message)
	return nil // Always succeed
}
