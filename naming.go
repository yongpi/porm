package porm

import (
	"unicode"
)

type NamingFunc func(string) string

var DefaultNamingFunc = HumpNamingFunc

func HumpNamingFunc(name string) string {
	var ui int
	var result []rune
	for index, value := range []rune(name) {
		isUpper := unicode.IsUpper(value)
		if isUpper {
			if index-ui > 1 {
				result = append(result, '_')
			}
			ui = index
			result = append(result, unicode.ToLower(value))
			continue
		}

		result = append(result, value)
	}

	return string(result)
}
