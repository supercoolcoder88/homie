package homeassistant

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gorilla/websocket"
)

type Device struct {
	EntityID string
}

type Service struct {
	token   string
	conn    *websocket.Conn
	msgID   int
	Devices []Device
}

func NewService(token string) *Service {
	return &Service{
		token: token,
		msgID: 1,
	}
}

// connect establishes and authenticates the websocket connection.
func (s *Service) Connect() error {
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8123/api/websocket", nil)
	if err != nil {
		return fmt.Errorf("failed to connect to websocket: %w", err)
	}

	// Read the auth_required message
	var msg map[string]interface{}
	if err := conn.ReadJSON(&msg); err != nil {
		conn.Close()
		return fmt.Errorf("failed to read auth_required: %w", err)
	}

	// Send auth message
	auth := map[string]string{
		"type":         "auth",
		"access_token": s.token,
	}
	if err := conn.WriteJSON(auth); err != nil {
		conn.Close()
		return fmt.Errorf("failed to send auth: %w", err)
	}

	// Read auth response
	if err := conn.ReadJSON(&msg); err != nil {
		conn.Close()
		return fmt.Errorf("failed to read auth response: %w", err)
	}
	if msg["type"] != "auth_ok" {
		conn.Close()
		return fmt.Errorf("auth failed: %v", msg)
	}

	s.conn = conn
	return nil
}

// Close closes the underlying websocket connection.
func (s *Service) Close() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

// GetDevices fetches the entity registry from Home Assistant
// using the persistent websocket connection.
func (s *Service) GetDevices() error {
	req := map[string]interface{}{
		"id":   s.msgID,
		"type": "config/entity_registry/list",
	}
	s.msgID++

	if err := s.conn.WriteJSON(req); err != nil {
		return fmt.Errorf("failed to request entity registry: %w", err)
	}

	_, rawMsg, err := s.conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var response struct {
		Result []struct {
			EntityID string `json:"entity_id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(rawMsg, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	devices := make([]Device, 0, len(response.Result))
	for _, entity := range response.Result {
		// eg: entity_id = "switch.tapo_p100_1"
		devices = append(devices, Device{
			EntityID: entity.EntityID,
		})
	}

	s.Devices = devices
	return nil
}

// ToggleEntities calls a turn_on or turn_off service for the given entity
// using the persistent websocket connection.
func (s *Service) ToggleEntities(entityIDs []string, newState string) error {
	if newState != "on" && newState != "off" {
		return fmt.Errorf("action must be 'on' or 'off', got '%s'", newState)
	}

	for _, entityID := range entityIDs {
		domain := strings.SplitN(entityID, ".", 2)[0]
		svc := "turn_" + newState

		req := map[string]interface{}{
			"id":      s.msgID,
			"type":    "call_service",
			"domain":  domain,
			"service": svc,
			"service_data": map[string]interface{}{
				"entity_id": entityID,
			},
		}
		s.msgID++

		if err := s.conn.WriteJSON(req); err != nil {
			return fmt.Errorf("failed to send service call: %w", err)
		}

		_, _, err := s.conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

	}

	// Note: response is not handled
	return nil
}
