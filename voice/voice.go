package voice

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

const defaultOutputFile = "/tmp/homie_recording.wav"

// Record captures audio from the system microphone using arecord (ALSA).
// It records 16-bit mono PCM at 16kHz (the format Whisper expects).
// The user presses Enter to stop recording.
// Returns the path to the recorded WAV file.
func Record() (string, error) {
	outputPath := defaultOutputFile

	// Remove any previous recording.
	os.Remove(outputPath)

	// arecord flags:
	//   -f S16_LE  : 16-bit signed little-endian
	//   -r 16000   : 16 kHz sample rate
	//   -c 1       : mono
	//   -t wav     : WAV container
	cmd := exec.Command("arecord", "-f", "S16_LE", "-r", "16000", "-c", "1", "-t", "wav", outputPath)
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start recording (is arecord installed?): %w", err)
	}

	log.Println("Recording... Press Enter to stop.")

	// Block until the user presses Enter.
	buf := make([]byte, 1)
	os.Stdin.Read(buf)

	// Send interrupt to stop arecord gracefully so it finalises the WAV header.
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		// If interrupt fails, kill it.
		cmd.Process.Kill()
	}
	cmd.Wait()

	// Verify the file was created.
	if _, err := os.Stat(outputPath); err != nil {
		return "", fmt.Errorf("recording file not found: %w", err)
	}

	log.Println("Recording saved to", outputPath)
	return outputPath, nil
}
