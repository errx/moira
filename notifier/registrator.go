package notifier

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"

	// "git.skbkontur.ru/devops/kontur"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/senders/mail"
	"github.com/moira-alert/moira/senders/pushover"
	"github.com/moira-alert/moira/senders/script"
	"github.com/moira-alert/moira/senders/slack"
	"github.com/moira-alert/moira/senders/telegram"
	"github.com/moira-alert/moira/senders/twilio_sms"
	"github.com/moira-alert/moira/senders/twilio_voice"
	
)

// RegisterSenders watch on senders config and register all configured senders
func (notifier *StandardNotifier) RegisterSenders(connector moira.Database, frontURI string) error {
	for _, rawSettings := range notifier.config.Senders {
		var settings moira.SenderBaseConfig
		if err := mapstructure.Decode(rawSettings, &settings); err != nil {
			return fmt.Errorf("Can't parse sender settings: %v", err)
		}
		if settings.Name == "" || settings.Type == "" {
			return fmt.Errorf("Can't parse sender settings: type or name is empty")
		}
		notifier.logger.Infof("Registering sender name %s type %s", settings.Name, settings.Type)
		settings.FrontURI = frontURI
		var err error 
		switch settings.Type {  
		case "pushover":
			err = notifier.RegisterSender(settings.Name, rawSettings, &pushover.Sender{})
		case "slack":
			err = notifier.RegisterSender(settings.Name, rawSettings, &slack.Sender{})
		case "mail":
			err = notifier.RegisterSender(settings.Name, rawSettings, &mail.Sender{})
		case "script":
			err = notifier.RegisterSender(settings.Name, rawSettings, &script.Sender{})
		case "telegram":
			err = notifier.RegisterSender(settings.Name, rawSettings, &telegram.Sender{DataBase: connector})
		case "twilio sms":
			err = notifier.RegisterSender(settings.Name, rawSettings, &twilio_sms.Sender{})
		case "twilio voice":
			err = notifier.RegisterSender(settings.Name, rawSettings, &twilio_voice.Sender{})
		// case "email":
		// 	err = notifier.RegisterSender(settings.Name, senderSettings, &kontur.MailSender{})
		// case "phone":
		// 	err = notifier.RegisterSender(settings.Name, senderSettings, &kontur.SmsSender{})
		default:
			return fmt.Errorf("Unknown type [%s] for sender %s", settings.Type, settings.Name)
		}
		if err != nil {
			return fmt.Errorf("Can not register sender %s: %s", settings.Name, err)
		}
	}
	return nil
}

// RegisterSender adds sender for notification type and registers metrics
func (notifier *StandardNotifier) RegisterSender(senderIdent string, senderSettings interface{}, sender moira.Sender) error {
	if err := sender.Init(senderSettings, notifier.logger); err != nil {
		return fmt.Errorf("Don't initialize sender [%s], err [%v]", senderIdent, err)
	}
	ch := make(chan NotificationPackage)
	notifier.senders[senderIdent] = ch
	notifier.metrics.SendersOkMetrics.AddMetric(senderIdent, fmt.Sprintf("notifier.%s.sends_ok", getGraphiteSenderIdent(senderIdent)))
	notifier.metrics.SendersFailedMetrics.AddMetric(senderIdent, fmt.Sprintf("notifier.%s.sends_failed", getGraphiteSenderIdent(senderIdent)))
	notifier.waitGroup.Add(1)
	go notifier.run(sender, ch)
	notifier.logger.Debugf("Sender %s registered", senderIdent)
	return nil
}

// StopSenders close all sending channels
func (notifier *StandardNotifier) StopSenders() {
	for _, ch := range notifier.senders {
		close(ch)
	}
	notifier.senders = make(map[string]chan NotificationPackage)
	notifier.logger.Debug("Waiting senders finish ...")
	notifier.waitGroup.Wait()
}

func getGraphiteSenderIdent(ident string) string {
	return strings.Replace(ident, " ", "_", -1)
}
