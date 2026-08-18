package ansi

import "unicode/utf8"

func decodeRune(s string) (rune, int) {
	if s == "" {
		return 0, 0
	}
	if s[0] < utf8.RuneSelf {
		return rune(s[0]), 1
	}
	return utf8.DecodeRuneInString(s)
}
