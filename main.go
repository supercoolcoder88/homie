package main

import (
	"fmt"
	"homie/homeassistant"
	"homie/ollama"
	"homie/voice"
	"homie/whisper"
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

	// Collect entity IDs for the LLM context
	entityIDs := make([]string, len(s.Devices))
	for i, d := range s.Devices {
		entityIDs[i] = d.EntityID
	}

	// Initialize the Ollama LLM client
	llm := ollama.NewClient("llama3.2")

	// Record voice input from the microphone.
	audioPath, err := voice.Record()
	if err != nil {
		log.Fatal("failed to record audio:", err)
	}

	// Transcribe the recorded audio using the local Whisper server.
	w := whisper.NewClient("http://localhost:8080")
	userInput, err := w.Transcribe(audioPath)
	if err != nil {
		log.Fatal("failed to transcribe audio:", err)
	}

	log.Printf("Transcribed: %s", userInput)

	cmd, err := llm.Interpret(userInput, entityIDs)
	if err != nil {
		log.Fatal("failed to interpret command:", err)
	}

	if err := HandleAction(s, cmd); err != nil {
		log.Fatal("failed to execute action")
	}
}

type ActionFunc func(*homeassistant.Service, *ollama.DeviceCommand) error

func HandleAction(service *homeassistant.Service, cmd *ollama.DeviceCommand) error {
	actionsMap := map[string]ActionFunc{
		"toggle_device": ToggleDevice,
	}

	// given llm response here
	fn, ok := actionsMap[cmd.Action]

	if !ok {
		return fmt.Errorf("action not found")
	}

	if err := fn(service, cmd); err != nil {
		return err
	}
	return nil
}

func ToggleDevice(service *homeassistant.Service, cmd *ollama.DeviceCommand) error {
	if len(cmd.EntityIDs) == 0 {
		return fmt.Errorf("entity_ids is empty")
	}

	if cmd.NewState == "" {
		return fmt.Errorf("newState is empty")
	}

	if err := service.ToggleEntities(cmd.EntityIDs, cmd.NewState); err != nil {
		return err
	}

	log.Printf("success, turned %s %s", cmd.EntityIDs, cmd.Action)
	return nil
}
