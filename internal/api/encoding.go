package api

import (
	"bytes"
	"io"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// decodeResponse decodes raw bytes returned by Tencent Finance endpoints.
//
// The endpoints inconsistently use UTF-8 or GB18030 (a superset of GBK) depending on
// the path. We try UTF-8 first to avoid corrupting already-valid UTF-8 payloads, then
// fall back to GB18030, and finally return the raw bytes as a last resort.
func decodeResponse(raw []byte) string {
	if utf8.Valid(raw) {
		return string(raw)
	}
	if decoded, ok := tryDecode(raw, simplifiedchinese.GB18030.NewDecoder()); ok {
		return decoded
	}
	return string(raw)
}

func tryDecode(raw []byte, decoder transform.Transformer) (string, bool) {
	reader := transform.NewReader(bytes.NewReader(raw), decoder)
	decoded, err := io.ReadAll(reader)
	if err != nil || !utf8.Valid(decoded) {
		return "", false
	}
	return string(decoded), true
}
