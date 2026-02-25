package main

import (
	"fmt"
	"homie/homeassistant"
	"log"
	"os"
	"strings"
)

func loadToken() string {
	data, err := os.ReadFile(".env")
	if err != nil {
		log.Fatal("failed to read .env file:", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "HOME_ASSISTANT_TOKEN=") {
			return strings.TrimPrefix(line, "HOME_ASSISTANT_TOKEN=")
		}
	}
	log.Fatal("HOME_ASSISTANT_TOKEN not found in .env")
	return ""
}

func main() {
	token := loadToken()

	s := homeassistant.NewService(token)

	if err := s.Connect(); err != nil {
		log.Fatal("failed to connect to homeassistant")
	}

	defer s.Close()

	if err := s.GetDevices(); err != nil {
		log.Fatal("failed to get devices")
	}
}

type ActionFunc func(homeassistant.Service, map[string]any) error

func HandleAction(service homeassistant.Service, action string, params map[string]any) error {
	actionsMap := map[string]ActionFunc{
		"toggle_device": ToggleDevice,
	}

	// given llm response here
	fn, ok := actionsMap[action]

	if !ok {
		return fmt.Errorf("action not found")
	}

	if err := fn(service, params); err != nil {
		return err

	}
	return nil
}

func ToggleDevice(service homeassistant.Service, params map[string]any) error {
	entityIDs, ok := params["entity_ids"].([]string)

	if !ok {
		return fmt.Errorf("entity_ids is missing from params")
	}

	newState, ok := params["newState"].(string)

	if !ok {
		return fmt.Errorf("newState is missing from params")
	}

	if err := service.ToggleEntities(entityIDs, newState); err != nil {
		return err
	}

	return nil
}
