package ai

import (
	"oqs"
	"os"
)

func EncryptFile(inputPath, outputPath string, key []byte) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}

	encryptedData, err := oqs.Encrypt(data, key)
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, encryptedData, 0644)
}

func DecryptFile(inputPath, outputPath string, key []byte) error {
	encryptedData, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}

	data, err := oqs.Decrypt(encryptedData, key)
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, data, 0644)
}
