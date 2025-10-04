package notification

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	mail "github.com/xhit/go-simple-mail/v2"
)

// smtpEmailSender is the concrete implementation for sending emails via SMTP.
type smtpEmailSender struct {
	client *mail.SMTPServer
	from   string
	log    *slog.Logger
}

// NewSMTPEmailSender creates a new sender that uses an SMTP server.
func NewSMTPEmailSender(host string, port int, username, password, from string, log *slog.Logger) emailSender {
	server := mail.NewSMTPClient()
	server.Host = host
	server.Port = port
	server.Username = username
	server.Password = password
	server.Encryption = mail.EncryptionSTARTTLS
	server.KeepAlive = false
	server.ConnectTimeout = 10 * time.Second
	server.SendTimeout = 10 * time.Second

	return &smtpEmailSender{
		client: server,
		from:   from,
		log:    log,
	}
}

func (s *smtpEmailSender) Send(ctx context.Context, to, subject, htmlBody string) error {
	smtpClient, err := s.client.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	email := mail.NewMSG()
	email.SetFrom(s.from).AddTo(to).SetSubject(subject)
	email.SetBody(mail.TextHTML, htmlBody)

	if err = email.Send(smtpClient); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	s.log.Info("email sent via smtp", "to", to)
	return nil
}
