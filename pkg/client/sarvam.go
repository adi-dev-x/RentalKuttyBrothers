package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"myproject/pkg/model"
	"net/http"
	"time"
)

const (
	sarvamSTTURL = "https://api.sarvam.ai/speech-to-text"
	sarvamTTSURL = "https://api.sarvam.ai/text-to-speech"

	// sarvamTTSMaxChars is the bulbul:v3 per-request text limit.
	sarvamTTSMaxChars = 2500
)

// SpeechLanguages are the languages Sarvam text-to-speech can speak, by language code.
var SpeechLanguages = map[string]string{
	"en-IN": "English",
	"hi-IN": "Hindi",
	"ml-IN": "Malayalam",
	"ta-IN": "Tamil",
	"te-IN": "Telugu",
	"kn-IN": "Kannada",
	"bn-IN": "Bengali",
	"gu-IN": "Gujarati",
	"mr-IN": "Marathi",
	"od-IN": "Odia",
	"pa-IN": "Punjabi",
}

var sarvamClient = &http.Client{Timeout: 60 * time.Second}

// TranscribeAudio sends audio to Sarvam and returns the transcript in the spoken language,
// with the detected language code.
func (s MyService) TranscribeAudio(audio []byte, filename string) (*model.TranscriptResponse, error) {
	apiKey := s.Config.SarvamAPIKey
	if apiKey == "" {
		return nil, fmt.Errorf("SARVAM_API_KEY is not set")
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	filePart, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	if _, err := filePart.Write(audio); err != nil {
		return nil, err
	}

	_ = writer.WriteField("model", "saaras:v4")
	_ = writer.WriteField("language_code", "unknown")

	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, sarvamSTTURL, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("api-subscription-key", apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := sarvamClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("Sarvam returned %d: %s", resp.StatusCode, string(message))
	}

	var result model.TranscriptResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// TextToSpeech converts text to speech in languageCode and returns base64-encoded WAV audio.
func (s MyService) TextToSpeech(text, languageCode string) (string, error) {
	apiKey := s.Config.SarvamAPIKey
	if apiKey == "" {
		return "", fmt.Errorf("SARVAM_API_KEY is not set")
	}
	if runes := []rune(text); len(runes) > sarvamTTSMaxChars {
		text = string(runes[:sarvamTTSMaxChars])
	}

	payload, err := json.Marshal(map[string]interface{}{
		"text":          text,
		"language_code": languageCode,
		"model":         "bulbul:v3",
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, sarvamTTSURL, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("api-subscription-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := sarvamClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("Sarvam TTS returned %d: %s", resp.StatusCode, string(message))
	}

	var result struct {
		Audios []string `json:"audios"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Audios) == 0 {
		return "", fmt.Errorf("Sarvam TTS returned no audio")
	}
	return result.Audios[0], nil
}
