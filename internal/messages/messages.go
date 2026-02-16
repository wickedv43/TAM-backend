package messages

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"

	"github.com/samber/do/v2"
	"github.com/wickedv43/TAM-backend/internal/config"
	"github.com/wickedv43/TAM-backend/internal/modules/telegram_bot/util"
	"gopkg.in/yaml.v3"
)

//go:embed bot_messages.yaml
var messagesFS embed.FS

type Messages struct {
	sections map[string]map[string]map[string]string // locale -> section -> key -> template
}

// New creates a new Messages instance from DI
func New(i do.Injector) (*Messages, error) {
	_ = do.MustInvoke[*config.Config](i) // For future use if we add locale to config
	return Load()
}

// Load loads messages from embedded YAML file
func Load() (*Messages, error) {
	data, err := messagesFS.ReadFile("bot_messages.yaml")
	if err != nil {
		return nil, fmt.Errorf("read bot_messages.yaml: %w", err)
	}

	var raw map[string]map[string]map[string]string
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal bot_messages.yaml: %w", err)
	}

	return &Messages{sections: raw}, nil
}

// Render renders a message template with the given data
func (m *Messages) Render(locale, section, key string, data interface{}) (string, error) {
	locale = m.resolveLocale(locale)

	sectionMap, ok := m.sections[locale]
	if !ok {
		return "", fmt.Errorf("locale %s not found", locale)
	}

	keyMap, ok := sectionMap[section]
	if !ok {
		return "", fmt.Errorf("section %s.%s not found", locale, section)
	}

	tplStr, ok := keyMap[key]
	if !ok {
		return "", fmt.Errorf("key %s.%s.%s not found", locale, section, key)
	}

	if data == nil {
		return tplStr, nil
	}

	tpl, err := template.New("").Parse(tplStr)
	if err != nil {
		return tplStr, fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err = tpl.Execute(&buf, data); err != nil {
		return tplStr, fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

// RenderOrDefault renders a message template, returning fallback on error or empty result
func (m *Messages) RenderOrDefault(locale, section, key string, data interface{}, fallback string) string {
	s, err := m.Render(locale, section, key, data)
	if err != nil || s == "" {
		return fallback
	}
	return s
}

// RenderMDV2 renders a message and escapes it for MarkdownV2
func (m *Messages) RenderMDV2(locale, section, key string, data interface{}) (string, error) {
	raw, err := m.Render(locale, section, key, data)
	if err != nil {
		return "", err
	}
	return util.EscapeMarkdownV2(raw), nil
}

// resolveLocale returns the locale with fallback to "en"
func (m *Messages) resolveLocale(locale string) string {
	if locale == "" {
		return "en"
	}
	if _, ok := m.sections[locale]; ok {
		return locale
	}
	return "en"
}

// WrapSpoiler wraps text in MarkdownV2 spoiler tags without escaping the delimiters
func WrapSpoiler(text string) string {
	return "||" + util.EscapeMarkdownV2(text) + "||"
}
