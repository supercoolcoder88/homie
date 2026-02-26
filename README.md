# homie

## Setup instructions
1. Download ollama
    a. ollama pull llama3.2
2. Download homeassistant 
    a. Need to name devices appropriately
3. Download model for whisper
curl -L -o models/ggml-base.en.bin "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.en.bin"


## When running
```
    ollama serve
    ALSA_DEVICE=default:CARD=Generic_1 go run main.go
```