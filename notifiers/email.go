package notifiers

import (
	"bytes"
	"context"
	"hawkeye/collector/aggregator"
	"hawkeye/utils"
	"log"
	"text/template"
	"time"
)

type Notifier interface {
	Send(context.Context, map[string]interface{}) error
}

type NotifierConfig struct {
	lastNotifyAt time.Time
	interval     time.Duration
	ServiceName  string
	Environment  string
}

type EmailNotifier struct {
	config    NotifierConfig
	mailer    MailingService
	mailerCfg MailerConfig
}

func NewEmailNotifier(mailer MailingService, trigger aggregator.Trigger, cfg NotifierConfig) EmailNotifier {
	var buf = bytes.Buffer{}

	buf.WriteString(cfg.ServiceName)
	buf.WriteString("\n")
	buf.WriteString("SLA breached in")
	buf.WriteString(" ")
	buf.WriteString(cfg.Environment)
	buf.WriteString("\n")

	if trigger.Text != nil {
		buf.WriteString(*trigger.Text)
		buf.WriteString("\n")
	}

	interval := -1 * time.Duration(trigger.RunEveryMinute) * time.Minute

	return EmailNotifier{
		mailer: mailer,
		mailerCfg: MailerConfig{
			Subject:    trigger.Subject,
			Body:       buf.String(),
			Recipients: trigger.To,
		},
		config: NotifierConfig{
			lastNotifyAt: utils.Now().Add(-interval),
			interval:     interval,
		},
	}
}

func (n EmailNotifier) Send(ctx context.Context, values map[string]interface{}) error {
	now := utils.Now()
	if now.Sub(n.config.lastNotifyAt) < n.config.interval {
		return nil
	}

	n.config.lastNotifyAt = now

	log.Println("SLA Breached. Notifying")
	// log.Println("sending emails ", body, n.subject)

	n.mailerCfg.Body = RenderTextTemplate(n.mailerCfg.Body, values)
	return n.mailer.Send(ctx, n.mailerCfg)
}

func RenderTextTemplate(text string, values map[string]interface{}) string {
	templ := template.Must(template.New("custom template").Parse(text))
	var by bytes.Buffer

	if err := templ.Execute(&by, values); err != nil {
		log.Println("failed to render template ", err)
		return text
	}

	return by.String()
}
