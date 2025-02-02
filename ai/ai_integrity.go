package ai

import (
	"fmt"
	"oqs"
	"os"
)

func SignModel(filePath string, privateKey []byte) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return oqs.Sign(data, privateKey)
}

func VerifyModel(filePath string, signature, publicKey []byte) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	if !oqs.VerifySignature(data, signature, publicKey) {
		return fmt.Errorf("signature verification failed for %s", filePath)
	}
	return nil
}
