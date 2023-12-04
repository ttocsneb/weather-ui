package util

import (
	"strconv"
	"strings"

	"github.com/oxtoacart/bpool"
)

var BufPool *bpool.BufferPool

func Setup() {
	BufPool = bpool.NewBufferPool(64)
}

func DecodeURIString(data string) (string, error) {
	data = strings.ReplaceAll(data, "+", " ")

	var builder strings.Builder

	i := 0
	for i < len(data) {
		if data[i] == '%' {
			var code string
			if data[i+1] == 'u' || data[i+1] == 'U' {
				code = data[i+2 : i+2+4]
				i += 6
			} else {
				code = data[i+1 : i+1+2]
				i += 3
			}
			val, err := strconv.ParseInt(code, 32, 0)
			if err != nil {
				return "", err
			}
			_, err = builder.WriteRune(rune(val))
			if err != nil {
				return "", err
			}
		} else {
			err := builder.WriteByte(data[i])
			if err != nil {
				return "", err
			}
			i += 1
		}
	}

	return builder.String(), nil
}

func EncodeURIString(data string) string {
	var builder strings.Builder

	for i := 0; i < len(data); i++ {
		c := data[i]
		switch {
		case c == ' ':
			builder.WriteRune('+')
		case (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') && (c < '0' || c > '9') &&
			c != '-' && c != '_' && c != '.' && c != '~':
			builder.WriteRune('%')
			builder.WriteString(strconv.FormatInt(int64(c), 16))
		default:
			builder.WriteByte(c)
		}
	}
	return builder.String()
}

func EscapeHtmlNewlines(data string) string {
	content := strings.ReplaceAll(data, "\n", " ")
	return strings.Join(strings.Fields(content), " ")
}

func Contains[T comparable](vals []T, val *T) bool {
	for _, v := range vals {
		if *val == v {
			return true
		}
	}
	return false
}
