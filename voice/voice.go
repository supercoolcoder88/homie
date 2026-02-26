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
// Records for 5 seconds then stops automatically.
// Returns the path to the recorded WAV file.
//
// Set the ALSA_DEVICE environment variable to override the capture device
// (e.g. "default:CARD=Generic_1"). If unset, the system default is used.
func Record() (string, error) {
	outputPath := defaultOutputFile

	os.Remove(outputPath)

	// arecord flags:
	//   -D device  : ALSA capture device (optional)
	//   -d 5       : record for 5 seconds
	//   -f S16_LE  : 16-bit signed little-endian
	//   -r 16000   : 16 kHz sample rate
	//   -c 1       : mono
	//   -t wav     : WAV container
	args := []string{}
	if dev := os.Getenv("ALSA_DEVICE"); dev != "" {
		args = append(args, "-D", dev)
	}
	args = append(args, "-d", "4", "-f", "S16_LE", "-r", "16000", "-c", "1", "-t", "wav", outputPath)
	cmd := exec.Command("arecord", args...)
	cmd.Stderr = os.Stderr

	log.Println("Recording for 4 seconds...")

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("recording failed: %w", err)
	}

	// Verify the file was created.
	if _, err := os.Stat(outputPath); err != nil {
		return "", fmt.Errorf("recording file not found: %w", err)
	}

	return outputPath, nil
}
