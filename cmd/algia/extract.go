package main

import (
	"regexp"
	"strings"
)

const (
	urlPattern     = `https?://[-A-Za-z0-9+&@#\/%?=~_|!:,.;\(\)]+`
	mentionPattern = `@[a-zA-Z0-9.]+`
	emojiPattern   = `:[a-zA-Z0-9]+:`
)

var (
	urlRe     = regexp.MustCompile(urlPattern)
	mentionRe = regexp.MustCompile(mentionPattern)
	emojiRe   = regexp.MustCompile(emojiPattern)
)

type matchLocation struct {
	start int64
	end   int64
	text  string
}

func extractLinks(text string) []matchLocation {
	var result []matchLocation
	matches := urlRe.FindAllStringSubmatchIndex(text, -1)
	for _, m := range matches {
		result = append(result, matchLocation{
			text:  text[m[0]:m[1]],
			start: int64(len([]rune(text[0:m[0]]))),
			end:   int64(len([]rune(text[0:m[1]])))},
		)
	}
	return result
}

func extractMentions(text string) []matchLocation {
	var result []matchLocation
	matches := mentionRe.FindAllStringSubmatchIndex(text, -1)
	for _, m := range matches {
		result = append(result, matchLocation{
			text:  strings.TrimPrefix(text[m[0]:m[1]], "@"),
			start: int64(len([]rune(text[0:m[0]]))),
			end:   int64(len([]rune(text[0:m[1]])))},
		)
	}
	return result
}

func extractEmojis(text string) []matchLocation {
	var result []matchLocation
	matches := emojiRe.FindAllStringSubmatchIndex(text, -1)
	for _, m := range matches {
		result = append(result, matchLocation{
			text:  text[m[0]:m[1]],
			start: int64(len([]rune(text[0:m[0]]))),
			end:   int64(len([]rune(text[0:m[1]])))},
		)
	}
	return result
}
