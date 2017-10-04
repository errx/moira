package slack

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/moira-alert/moira"
	"github.com/nlopes/slack"
)

type Config struct {
	APIToken               string `mapstructure:"api_token"`
	moira.SenderBaseConfig `mapstructure:",squash"`
}

// Sender implements moira sender interface via slack
type Sender struct {
	config Config
	log    moira.Logger
}

// Init read yaml config
func (sender *Sender) Init(senderSettings interface{}, logger moira.Logger) error {
	sender.log = logger
	if err := mapstructure.Decode(senderSettings, sender.config); err != nil {
		return err
	}
	if sender.config.APIToken == "" {
		return fmt.Errorf("Can not read slack api_token from config")
	}

	if _, err := slack.New(sender.config.APIToken).AuthTest(); err != nil {
		return err
	}

	return nil
}

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, throttled bool) error {
	api := slack.New(sender.config.APIToken)
	var message bytes.Buffer
	state := events.GetSubjectState()
	tags := trigger.GetTags()
	message.WriteString(fmt.Sprintf("*%s* %s <%s/#/events/%s|%s>\n %s \n```", state, tags, sender.config.FrontURI, events[0].TriggerID, trigger.Name, trigger.Desc))
	icon := fmt.Sprintf("%s/public/fav72_ok.png", sender.config.FrontURI)
	for _, event := range events {
		if event.State != "OK" {
			icon = fmt.Sprintf("%s/public/fav72_error.png", sender.config.FrontURI)
		}
		value := strconv.FormatFloat(moira.UseFloat64(event.Value), 'f', -1, 64)
		message.WriteString(fmt.Sprintf("\n%s: %s = %s (%s to %s)", time.Unix(event.Timestamp, 0).Format("15:04"), event.Metric, value, event.OldState, event.State))
		if len(moira.UseString(event.Message)) > 0 {
			message.WriteString(fmt.Sprintf(". %s", moira.UseString(event.Message)))
		}
	}

	message.WriteString("```")

	if throttled {
		message.WriteString("\nPlease, *fix your system or tune this trigger* to generate less events.")
	}

	sender.log.Debugf("Calling slack with message body %s", message.String())

	params := slack.PostMessageParameters{
		Username: "Moira",
		IconURL:  icon,
	}

	_, _, err := api.PostMessage(contact.Value, message.String(), params)
	if err != nil {
		return fmt.Errorf("Failed to send message to slack [%s]: %s", contact.Value, err.Error())
	}
	return nil
}
