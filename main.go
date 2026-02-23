package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gorilla/websocket"
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

func getDevices(token string) (string, error) {
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8123/api/websocket", nil)
	if err != nil {
		return "", fmt.Errorf("failed to connect to websocket: %w", err)
	}
	defer conn.Close()

	// Read the auth_required message
	var msg map[string]interface{}
	if err := conn.ReadJSON(&msg); err != nil {
		return "", fmt.Errorf("failed to read auth_required: %w", err)
	}

	// Send auth message
	auth := map[string]string{
		"type":         "auth",
		"access_token": token,
	}
	if err := conn.WriteJSON(auth); err != nil {
		return "", fmt.Errorf("failed to send auth: %w", err)
	}

	// Read auth response
	if err := conn.ReadJSON(&msg); err != nil {
		return "", fmt.Errorf("failed to read auth response: %w", err)
	}
	if msg["type"] != "auth_ok" {
		return "", fmt.Errorf("auth failed: %v", msg)
	}

	// Request entity registry
	req := map[string]interface{}{
		"id":   1,
		"type": "config/entity_registry/list",
	}
	if err := conn.WriteJSON(req); err != nil {
		return "", fmt.Errorf("failed to request entity registry: %w", err)
	}

	// Read response
	_, rawMsg, err := conn.ReadMessage()
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Pretty print the JSON
	var parsed interface{}
	if err := json.Unmarshal(rawMsg, &parsed); err != nil {
		return string(rawMsg), nil
	}
	pretty, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return string(rawMsg), nil
	}

	return string(pretty), nil
}

func toggleEntity(token string, entityID string, action string) (string, error) {
	if action != "on" && action != "off" {
		return "", fmt.Errorf("action must be 'on' or 'off', got '%s'", action)
	}

	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8123/api/websocket", nil)
	if err != nil {
		return "", fmt.Errorf("failed to connect to websocket: %w", err)
	}
	defer conn.Close()

	// Read the auth_required message
	var msg map[string]interface{}
	if err := conn.ReadJSON(&msg); err != nil {
		return "", fmt.Errorf("failed to read auth_required: %w", err)
	}

	// Send auth message
	auth := map[string]string{
		"type":         "auth",
		"access_token": token,
	}
	if err := conn.WriteJSON(auth); err != nil {
		return "", fmt.Errorf("failed to send auth: %w", err)
	}

	// Read auth response
	if err := conn.ReadJSON(&msg); err != nil {
		return "", fmt.Errorf("failed to read auth response: %w", err)
	}
	if msg["type"] != "auth_ok" {
		return "", fmt.Errorf("auth failed: %v", msg)
	}

	// Determine the service domain from the entity ID (e.g. "light" from "light.living_room")
	domain := strings.SplitN(entityID, ".", 2)[0]
	service := "turn_" + action

	req := map[string]interface{}{
		"id":      1,
		"type":    "call_service",
		"domain":  domain,
		"service": service,
		"service_data": map[string]interface{}{
			"entity_id": entityID,
		},
	}
	if err := conn.WriteJSON(req); err != nil {
		return "", fmt.Errorf("failed to send service call: %w", err)
	}

	// Read response
	_, rawMsg, err := conn.ReadMessage()
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var parsed interface{}
	if err := json.Unmarshal(rawMsg, &parsed); err != nil {
		return string(rawMsg), nil
	}
	pretty, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return string(rawMsg), nil
	}

	return string(pretty), nil
}

func main() {
	token := loadToken()
	args := os.Args[1:]

	if len(args) == 2 {
		// Usage: go run main.go <entity_id> <on|off>
		entityID := args[0]
		action := args[1]

		result, err := toggleEntity(token, entityID, action)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(result)
	} else {
		// No args: list all entities
		result, err := getDevices(token)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(result)
	}
}
