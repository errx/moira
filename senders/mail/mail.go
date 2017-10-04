package mail

import (
	"crypto/tls"
	"fmt"
	"html/template"
	"io"
	"net/smtp"
	"strconv"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/moira-alert/moira"
	gomail "gopkg.in/gomail.v2"
)

// Config
type Config struct {
	From                   string `mapstructure:"mail_from"`
	SMTPHost               string `mapstructure:"smtp_host"`
	SMTPPort               int  `mapstructure:"smtp_port"`
	InsecureTLS            bool   `mapstructure:"insecure_tls"`
	Password               string `mapstructure:"smtp_pass"`
	Username               string `mapstructure:"smtp_user"`
	TemplateFile           string `mapstructure:"template_file"`
	moira.SenderBaseConfig `mapstructure:",squash"`
}

// Sender implements moira sender interface via email
type Sender struct {
	config   *Config
	log      moira.Logger
	template *template.Template
}

type templateRow struct {
	Metric     string
	Timestamp  string
	Oldstate   string
	State      string
	Value      string
	WarnValue  string
	ErrorValue string
	Message    string
}

// Init read yaml config
func (sender *Sender) Init(senderSettings interface{}, logger moira.Logger) error {
	sender.log = logger
	if err := mapstructure.Decode(senderSettings, sender.config); err != nil {
		return err
	}

	if sender.config.Username == "" {
		sender.config.Username = sender.config.From
	}
	if sender.config.From == "" {
		return fmt.Errorf("mail_from can't be empty")
	}

	if sender.config.TemplateFile == "" {
		sender.template = template.Must(template.New("mail").Parse(defaultTemplate))
	} else {
		var err error
		if sender.template, err = template.New("mail").ParseFiles(sender.config.TemplateFile); err != nil {
			return err
		}
	}
	// Test connection
	t, err := smtp.Dial(fmt.Sprintf("%s:%d", sender.config.SMTPHost, sender.config.SMTPPort))
	if err != nil {
		return err
	}
	defer t.Close()
	// Test TLS handshake
	if err := t.StartTLS(&tls.Config{
		InsecureSkipVerify: sender.config.InsecureTLS,
		ServerName:         sender.config.SMTPHost,
	}); err != nil {
		return err
	}
	// Test authentication
	if sender.config.Password != "" {
		if err := t.Auth(smtp.PlainAuth(
			"",
			sender.config.Username,
			sender.config.Password,
			sender.config.SMTPHost,
		)); err != nil {
			return err
		}
	}
	return nil
}

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, throttled bool) error {

	m := sender.makeMessage(events, contact, trigger, throttled)

	d := gomail.Dialer{
		Host: sender.config.SMTPHost,
		Port: sender.config.SMTPPort,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: sender.config.InsecureTLS,
			ServerName:         sender.config.SMTPHost,
		},
	}

	if sender.config.Password != "" {
		d.Auth = smtp.PlainAuth(
			"", 
			sender.config.Username, 
			sender.config.Password, 
			sender.config.SMTPHost)
	}
	if err := d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}

func (sender *Sender) makeMessage(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, throttled bool) *gomail.Message {
	state := events.GetSubjectState()
	tags := trigger.GetTags()

	subject := fmt.Sprintf("%s %s %s (%d)", state, trigger.Name, tags, len(events))

	templateData := struct {
		Link        string
		Description string
		Throttled   bool
		Items       []*templateRow
	}{
		Link:        fmt.Sprintf("%s/#/events/%s", sender.config.FrontURI, events[0].TriggerID),
		Description: trigger.Desc,
		Throttled:   throttled,
		Items:       make([]*templateRow, 0, len(events)),
	}

	for _, event := range events {
		templateData.Items = append(templateData.Items, &templateRow{
			Metric:     event.Metric,
			Timestamp:  time.Unix(event.Timestamp, 0).Format("15:04 02.01.2006"),
			Oldstate:   event.OldState,
			State:      event.State,
			Value:      strconv.FormatFloat(moira.UseFloat64(event.Value), 'f', -1, 64),
			WarnValue:  strconv.FormatFloat(trigger.WarnValue, 'f', -1, 64),
			ErrorValue: strconv.FormatFloat(trigger.ErrorValue, 'f', -1, 64),
			Message:    moira.UseString(event.Message),
		})
	}

	m := gomail.NewMessage()
	m.SetHeader("From", sender.config.From)
	m.SetHeader("To", contact.Value)
	m.SetHeader("Subject", subject)
	m.AddAlternativeWriter("text/html", func(w io.Writer) error {
		return sender.template.Execute(w, templateData)
	})

	return m
}
