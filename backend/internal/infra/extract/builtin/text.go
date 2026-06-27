package builtin

import (
	"bytes"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// ExtractText 直接返回文本文件内容。
func ExtractText(data []byte) string {
	return strings.TrimSpace(decodeTextBytes(data))
}

func decodeTextBytes(data []byte) string {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	if utf8.Valid(data) {
		return string(data)
	}
	if text, _, err := transform.String(simplifiedchinese.GB18030.NewDecoder(), string(data)); err == nil && strings.TrimSpace(text) != "" {
		return text
	}
	return string(data)
}
