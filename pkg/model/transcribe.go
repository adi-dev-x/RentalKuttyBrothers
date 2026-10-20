package model

type TranscriptResponse struct {
	Transcript   string `json:"transcript"`
	LanguageCode string `json:"language_code"`
}
