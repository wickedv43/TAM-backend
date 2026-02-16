package util

import (
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v4"
)

var mdEscaper = strings.NewReplacer(
	`_`, `\_`,
	`[`, `\[`,
	`]`, `\]`,
	`(`, `\(`,
	`)`, `\)`,
	`~`, `\~`,
	"`", "\\`",
	`>`, `\>`,
	`#`, `\#`,
	`+`, `\+`,
	`-`, "\\-",
	`=`, `\=`,
	`|`, `\|`,
	`{`, `\{`,
	`}`, `\}`,
	`.`, `\.`,
	`!`, `\!`,
)

// Escape escapes characters for MarkdownV2.
func Escape(s string) string {
	return mdEscaper.Replace(s)
}

// EscapeMarkdownV2 escapes characters for MarkdownV2.
func EscapeMarkdownV2(s string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(s)
}

// utf16OffsetToRuneIndex converts a UTF-16 offset to a rune index.
func utf16OffsetToRuneIndex(runes []rune, utf16Offset int) int {
	currentUTF16 := 0
	for i, r := range runes {
		if currentUTF16 == utf16Offset {
			return i
		}
		if currentUTF16 > utf16Offset {
			return i
		}
		if r <= 0xFFFF {
			currentUTF16 += 1
		} else {
			currentUTF16 += 2
		}
	}
	if currentUTF16 == utf16Offset {
		return len(runes)
	}
	return len(runes)
}

// RenderMarkdownV2 converts a text with entities into a MarkdownV2 string.
func RenderMarkdownV2(text string, entities []tele.MessageEntity) string {
	if text == "" {
		return ""
	}

	runes := []rune(text)
	var builder strings.Builder

	type processedEntity struct {
		Type     tele.EntityType
		Start    int
		End      int
		URL      string
		User     *tele.User
		Lang     string
		CustomID string
	}

	var procEntities []processedEntity
	for _, e := range entities {
		startRune := utf16OffsetToRuneIndex(runes, e.Offset)
		endRune := utf16OffsetToRuneIndex(runes, e.Offset+e.Length)

		if startRune >= len(runes) {
			continue
		}
		if endRune > len(runes) {
			endRune = len(runes)
		}

		procEntities = append(procEntities, processedEntity{
			Type:     e.Type,
			Start:    startRune,
			End:      endRune,
			URL:      e.URL,
			User:     e.User,
			Lang:     e.Language,
			CustomID: e.CustomEmojiID,
		})
	}

	offset := 0
	for _, entity := range procEntities {
		if entity.Start < offset {
			continue
		}

		if entity.Start > offset {
			builder.WriteString(EscapeMarkdownV2(string(runes[offset:entity.Start])))
		}

		content := string(runes[entity.Start:entity.End])
		escapedContent := EscapeMarkdownV2(content)

		switch entity.Type {
		case tele.EntityBold:
			builder.WriteString("*" + escapedContent + "*")
		case tele.EntityItalic:
			builder.WriteString("_" + escapedContent + "_")
		case tele.EntityUnderline:
			builder.WriteString("__" + escapedContent + "__")
		case tele.EntityStrikethrough:
			builder.WriteString("~" + escapedContent + "~")
		case tele.EntityCode:
			builder.WriteString("`" + escapedContent + "`")
		case tele.EntityCodeBlock:
			builder.WriteString("```" + escapedContent + "```")
		case tele.EntityTextLink:
			builder.WriteString("[" + escapedContent + "](" + entity.URL + ")")
		case tele.EntityURL:
			builder.WriteString(escapedContent)
		case tele.EntityTMention:
			userLink := fmt.Sprintf("tg://user?id=%d", entity.User.ID)
			builder.WriteString("[" + escapedContent + "](" + userLink + ")")
		case tele.EntityCustomEmoji:
			builder.WriteString("![" + escapedContent + "](tg://emoji?id=" + entity.CustomID + ")")
		case tele.EntitySpoiler:
			builder.WriteString("||" + escapedContent + "||")
		default:
			builder.WriteString(escapedContent)
		}

		offset = entity.End
	}

	if offset < len(runes) {
		builder.WriteString(EscapeMarkdownV2(string(runes[offset:])))
	}

	return builder.String()
}
