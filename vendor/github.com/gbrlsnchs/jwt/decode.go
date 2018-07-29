package jwt

import "encoding/base64"

func decode(s string) ([]byte, error) {
	return base64.URLEncoding.
		WithPadding(base64.NoPadding).
		DecodeString(s)
}
