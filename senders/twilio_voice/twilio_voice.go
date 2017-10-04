package twilio_voice

import (
	"fmt"
	"net/url"

	twilio "github.com/carlosdp/twiliogo"
	"github.com/mitchellh/mapstructure"
	"github.com/moira-alert/moira"
)

type Config struct {
	APIASID                string `mapstructure:"api_asid"`
	APIAuthToken           string `mapstructure:"api_authtoken"`
	APIFromPhone           string `mapstructure:"api_fromphone"`
	VoiceURL               string `mapstructure:"voiceurl"`
	AppendMessage          bool   `mapstructure:"append_message"`
	moira.SenderBaseConfig `mapstructure:",squash"`
}

// Sender implements moira sender interface via twilio
type Sender struct {
	config Config
	log    moira.Logger
	client *twilio.TwilioClient
}

// Init read yaml config
func (sender *Sender) Init(senderSettings interface{}, logger moira.Logger) error {
	sender.log = logger
	if err := mapstructure.Decode(senderSettings, sender.config); err != nil {
		return err
	}
	if sender.config.APIASID == "" {
		return fmt.Errorf("Can't read 'api_sid' param for sender %s", sender.config.Name)
	}
	if sender.config.APIAuthToken == "" {
		return fmt.Errorf("Can't read 'api_authtoken' param for sender %s", sender.config.Name)
	}
	if sender.config.APIFromPhone == "" {
		return fmt.Errorf("Can't read 'api_fromphone' param for sender %s", sender.config.Name)
	}
	if sender.config.VoiceURL == "" {
		return fmt.Errorf("Can't read 'voiceurl' param for sender %s", sender.config.Name)
	}
	sender.client = twilio.NewClient(sender.config.APIASID, sender.config.APIAuthToken)
	// TODO: Here should be test connection and auth

	return nil
}

func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, throttled bool) error {
	voiceURL := sender.config.VoiceURL
	if sender.config.AppendMessage {
		voiceURL += url.QueryEscape(fmt.Sprintf("Hi! This is a notification for Moira trigger %s. Please, visit Moira web interface for details.", trigger.Name))
	}
	twilioCall, err := twilio.NewCall(sender.client, sender.config.APIFromPhone, contact.Value, twilio.Callback(voiceURL))
	if err != nil {
		return fmt.Errorf("Failed to make call to contact %s: %v", contact.Value, err)
	}
	sender.log.Debugf("Call queued to twilio with status %s, callback url %s", twilioCall.Status, voiceURL)

	return nil
}
