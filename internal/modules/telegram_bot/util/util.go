package util

import (
	"strings"

	tele "gopkg.in/telebot.v4"
)

var (
	OptTextMardown = &tele.SendOptions{ParseMode: tele.ModeMarkdownV2}
)

func Bold(s string) string {
	return "*" + s + "*"
}

func Italic(s string) string {
	return "_" + s + "_"
}

func Quote(s string) string {
	quoted := strings.Replace(s, "\n", "\n<", -1)
	return ">" + quoted
}

func Mono(s string) string {
	return "`" + s + "`"
}

func BoldHTML(s string) string {
	return "<b>" + s + "</b>"
}

func ItalicHTML(s string) string {
	return "<i>" + s + "</i>"
}

func StrikethroughHTML(s string) string {
	return "<s>" + s + "</s>"
}

func UnderlineHTML(s string) string {
	return "<u>" + s + "</u>"
}
