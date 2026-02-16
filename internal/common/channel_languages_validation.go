package common

import (
	_ "embed"
	"encoding/json"
)

//go:embed telegram-languages.json
var languagesJSON []byte

type TelegramLanguage struct {
	Code       string   `json:"code"`
	Name       string   `json:"name"`
	NativeName string   `json:"nativeName"`
	Official   bool     `json:"official"`
	RTL        bool     `json:"rtl"`
	Beta       bool     `json:"beta"`
	Platforms  []string `json:"platforms"`
}

var (
	languages     []TelegramLanguage
	languageNames map[string]bool
)

func init() {
	json.Unmarshal(languagesJSON, &languages) //nolint:errcheck

	languageNames = make(map[string]bool, len(languages))
	for _, lang := range languages {
		languageNames[lang.Code] = true
	}
}

func IsValidLanguage(language string) bool {
	return languageNames[language]
}
