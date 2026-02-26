package whisper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
)

// Client communicates with a local Whisper-compatible server
// (e.g. whisper.cpp server) to transcribe audio to text.
type Client struct {
	BaseURL string
}

// NewClient creates a new Whisper client pointing at the given server URL.
func NewClient(baseURL string) *Client {
	return &Client{BaseURL: baseURL}
}

// whisperResponse represents the JSON response from the Whisper server.
type whisperResponse struct {
	Text string `json:"text"`
}

// Transcribe sends an audio file to the Whisper server and returns the
// transcribed text as a string.
func (c *Client) Transcribe(audioPath string) (string, error) {
	f, err := os.Open(audioPath)
	if err != nil {
		return "", fmt.Errorf("failed to open audio file: %w", err)
	}
	defer f.Close()

	// Build a multipart form with the audio file.
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", audioPath)
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, f); err != nil {
		return "", fmt.Errorf("failed to copy audio data: %w", err)
	}

	// Close the writer to finalize the multipart body.
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	url := c.BaseURL + "/inference"
	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("whisper request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("whisper server returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result whisperResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode whisper response: %w", err)
	}

	return result.Text, nil
}
