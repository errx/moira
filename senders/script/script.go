package script

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/moira-alert/moira"
)

type Config struct {
	Exec                   string `mapstructure:"exec"`
	moira.SenderBaseConfig `mapstructure:",squash"`
}

// Sender implements moira sender interface via script execution
type Sender struct {
	config Config
	log    moira.Logger
}

type scriptNotification struct {
	Events    []moira.NotificationEvent `json:"events"`
	Trigger   moira.TriggerData         `json:"trigger"`
	Contact   moira.ContactData         `json:"contact"`
	Throttled bool                      `json:"throttled"`
	Timestamp int64                     `json:"timestamp"`
}

// Init read yaml config
func (sender *Sender) Init(senderSettings interface{}, logger moira.Logger) error {
	sender.log = logger
	if err := mapstructure.Decode(senderSettings, sender.config); err != nil {
		return err
	}
	if sender.config.Exec == "" {
		return fmt.Errorf("exec field cannot be empty for script type")
	}
	args := strings.Split(sender.config.Exec, " ")
	scriptFile := args[0]
	infoFile, err := os.Stat(scriptFile)
	if err != nil {
		return fmt.Errorf("File %s not found", scriptFile)
	}
	if !infoFile.Mode().IsRegular() {
		return fmt.Errorf("%s is not a file", scriptFile)
	}
	return nil
}

// SendEvents implements Sender interface Send
func (sender *Sender) SendEvents(events moira.NotificationEvents, contact moira.ContactData, trigger moira.TriggerData, throttled bool) error {
	execString := strings.Replace(sender.config.Exec, "${trigger_name}", trigger.Name, -1)
	execString = strings.Replace(execString, "${contact_value}", contact.Value, -1)

	args := strings.Split(execString, " ")
	scriptFile := args[0]
	infoFile, err := os.Stat(scriptFile)
	if err != nil {
		return fmt.Errorf("File %s not found", scriptFile)
	}
	if !infoFile.Mode().IsRegular() {
		return fmt.Errorf("%s is not a file", scriptFile)
	}

	scriptMessage := &scriptNotification{
		Events:    events,
		Trigger:   trigger,
		Contact:   contact,
		Throttled: throttled,
	}
	scriptJSON, err := json.MarshalIndent(scriptMessage, "", "\t")
	if err != nil {
		return fmt.Errorf("Failed marshal json")
	}

	c := exec.Command(scriptFile, args[1:]...)
	var scriptOutput bytes.Buffer
	c.Stdin = bytes.NewReader(scriptJSON)
	c.Stdout = &scriptOutput
	sender.log.Debugf("Executing script: %s", scriptFile)
	err = c.Run()
	sender.log.Debugf("Finished executing: %s", scriptFile)

	if err != nil {
		return fmt.Errorf("Failed exec [%s] Error [%v] Output: [%s]", sender.config.Exec, err, scriptOutput.String())
	}
	return nil
}
