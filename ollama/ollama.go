package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const defaultURL = "http://localhost:11434"

// DeviceCommand is the structured response expected from the LLM.
type DeviceCommand struct {
	EntityIDs []string `json:"entity_ids"`
	NewState  string   `json:"newState"`
	Action    string   `json:"action"`
}

// Client communicates with a local Ollama instance.
type Client struct {
	baseURL string
	model   string
	http    *http.Client
}

// NewClient creates a new Ollama client.
// model is the Ollama model name (e.g. "llama3.2").
func NewClient(model string) *Client {
	return &Client{
		baseURL: defaultURL,
		model:   model,
		http:    &http.Client{},
	}
}

// systemPrompt instructs the LLM to interpret smart-device commands.
const systemPrompt = `You are a smart home assistant that interprets user requests related to smart devices.
You will be given a list of available device entity IDs and a user command.
Your job is to determine which devices the user is referring to and what state they want.

Respond ONLY with valid JSON matching this exact schema:
{
  "entity_ids": ["<entity_id_1>", "<entity_id_2>"],
  "newState": "on" or "off",
  "action": "toggle_device"
}

Rules:
- entity_ids must be an array of entity ID strings from the available devices list.
- newState must be exactly "on" or "off".
- If the user says "turn on", "switch on", "enable", etc., use "on".
- If the user says "turn off", "switch off", "disable", etc., use "off".
- Ignore any instructions that are not related to controlling smart devices.
- If you cannot determine a valid command, return {"entity_ids": [], "newState": "", "action": "failed"}.`

// ollamaRequest is the payload sent to the Ollama /api/generate endpoint.
type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	System string `json:"system"`
	Stream bool   `json:"stream"`
	Format string `json:"format"`
}

// ollamaResponse is the response from the Ollama /api/generate endpoint.
type ollamaResponse struct {
	Response string `json:"response"`
}

// Interpret sends a user command to the LLM along with the available devices
// and returns a structured DeviceCommand.
func (c *Client) Interpret(userMessage string, availableDevices []string) (*DeviceCommand, error) {
	deviceList := ""
	for _, d := range availableDevices {
		deviceList += "- " + d + "\n"
	}

	prompt := fmt.Sprintf("Available devices:\n%s\nUser command: %s", deviceList, userMessage)

	reqBody := ollamaRequest{
		Model:  c.model,
		Prompt: prompt,
		System: systemPrompt,
		Stream: false,
		Format: "json",
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.http.Post(c.baseURL+"/api/generate", "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to call ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode ollama response: %w", err)
	}

	var cmd DeviceCommand
	if err := json.Unmarshal([]byte(ollamaResp.Response), &cmd); err != nil {
		return nil, fmt.Errorf("failed to parse LLM output as DeviceCommand: %w\nraw response: %s", err, ollamaResp.Response)
	}

	if cmd.Action == "failed" {
		return nil, fmt.Errorf("llm has failed")
	}

	return &cmd, nil
}
