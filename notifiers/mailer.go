package notifiers

import (
	"context"
	"log"
)

type MailerConfig struct {
	Subject    string
	Body       string
	Recipients []string
	Sender     string
	CC         []string
	Bcc        []string
}

type MailingService interface {
	Send(ctx context.Context, cfg MailerConfig) error
}

type MockMailingService struct{}

func (m MockMailingService) Send(ctx context.Context, cfg MailerConfig) error {
	for _, to := range cfg.Recipients {
		log.Println("sending email to ", to, cfg.Body)
	}

	log.Println("emails sent to ", len(cfg.Recipients), " recipients")
	return nil
}
