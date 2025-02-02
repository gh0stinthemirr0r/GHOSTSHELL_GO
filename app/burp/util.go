package burp

import (
	"encoding/base64"
)

// EncodeBase64 encodes a string to Base64
func EncodeBase64(data string) string {
	return base64.StdEncoding.EncodeToString([]byte(data))
}

// DecodeBase64 decodes a Base64 string
func DecodeBase64(data string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}
