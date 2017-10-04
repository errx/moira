package twilio_sms

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	twilio "github.com/carlosdp/twiliogo"
	"github.com/mitchellh/mapstructure"
	"github.com/moira-alert/moira"
)

type Config struct {
	APIASID                string `mapstructure:"api_asid"`
	APIAuthToken           string `mapstructure:"api_authtoken"`
	APIFromPhone           string `mapstructure:"api_fromphone"`
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

	sender.client = twilio.NewClient(sender.config.APIASID, sender.config.APIAuthToken)
	// TODO: Here should be test connection and auth
	return nil
}

func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, throttled bool) error {
	state := events.GetSubjectState()
	tags := trigger.GetTags()
	var message bytes.Buffer
	message.WriteString(fmt.Sprintf("%s %s %s (%d)\n", state, trigger.Name, tags, len(events)))

	for _, event := range events {
		value := strconv.FormatFloat(moira.UseFloat64(event.Value), 'f', -1, 64)
		message.WriteString(fmt.Sprintf("\n%s: %s = %s (%s to %s)", time.Unix(event.Timestamp, 0).Format("15:04"), event.Metric, value, event.OldState, event.State))
		if len(moira.UseString(event.Message)) > 0 {
			message.WriteString(fmt.Sprintf(". %s", moira.UseString(event.Message)))
		}
	}
	if len(events) > 5 {
		message.WriteString(fmt.Sprintf("\n\n...and %d more events.", len(events)-5))
	}
	if throttled {
		message.WriteString("\n\nPlease, fix your system or tune this trigger to generate less events.")
	}
	sender.log.Debugf("Calling twilio sms api to phone %s and message body %s", contact.Value, message.String())
	twilioMessage, err := twilio.NewMessage(sender.client, sender.config.APIFromPhone, contact.Value, twilio.Body(message.String()))
	if err != nil {
		return fmt.Errorf("Failed to send message to contact %s: %v", contact.Value, err)
	}
	sender.log.Debugf("message send to twilio with status: %s", twilioMessage.Status)
	return nil
}
