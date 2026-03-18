package api

import (
	"crypto/aes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const defaultEncryptPublicKey = "MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCkecphb6vgsBx4LJknKKes-eyj7-RKQ3fikF5B67EObZ3t4moFZyMGuuJPiadYdaxvRqtxyblIlVM7omAasROtKRhtgKwwRxo2a6878qBhTgUVlsqugpI_7ZC9RmO2Rpmr8WzDeAapGANfHN5bVr7G7GYGwIrjvyxMrAVit_oM4wIDAQAB"
const defaultApiAppKey = "B0455FBE7AA0328DB57B59AA729F05D8"
const defaultAccessKey = "9grzgbmxdsp3arfmmgq347xjbza4ysps"

func randomWord(length int) (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	if length <= 0 {
		return "", errors.New("invalid random word length")
	}
	raw := make([]byte, length)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.Grow(length)
	for i := 0; i < length; i++ {
		sb.WriteByte(alphabet[int(raw[i])%len(alphabet)])
	}
	return sb.String(), nil
}

func padPKCS7(data []byte, blockSize int) []byte {
	padLen := blockSize - (len(data) % blockSize)
	if padLen == 0 {
		padLen = blockSize
	}
	padded := make([]byte, len(data)+padLen)
	copy(padded, data)
	for i := len(data); i < len(padded); i++ {
		padded[i] = byte(padLen)
	}
	return padded
}

func unpadPKCS7(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, errors.New("invalid PKCS7 data length")
	}
	padLen := int(data[len(data)-1])
	if padLen == 0 || padLen > blockSize || padLen > len(data) {
		return nil, errors.New("invalid PKCS7 padding")
	}
	for i := len(data) - padLen; i < len(data); i++ {
		if int(data[i]) != padLen {
			return nil, errors.New("invalid PKCS7 padding bytes")
		}
	}
	return data[:len(data)-padLen], nil
}

func encryptHex(plain string, key string) (string, error) {
	switch len(key) {
	case 16, 24, 32:
	default:
		return "", fmt.Errorf("AES key must be 16/24/32 chars, got %d", len(key))
	}
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	data := padPKCS7([]byte(plain), block.BlockSize())
	out := make([]byte, len(data))
	for bs := 0; bs < len(data); bs += block.BlockSize() {
		block.Encrypt(out[bs:bs+block.BlockSize()], data[bs:bs+block.BlockSize()])
	}
	return hex.EncodeToString(out), nil
}

func decryptHex(cipherHex string, key string) ([]byte, error) {
	switch len(key) {
	case 16, 24, 32:
	default:
		return nil, fmt.Errorf("AES key must be 16/24/32 chars, got %d", len(key))
	}
	raw, err := hex.DecodeString(strings.TrimSpace(cipherHex))
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}
	if len(raw)%block.BlockSize() != 0 {
		return nil, errors.New("invalid AES block length")
	}

	out := make([]byte, len(raw))
	for bs := 0; bs < len(raw); bs += block.BlockSize() {
		block.Decrypt(out[bs:bs+block.BlockSize()], raw[bs:bs+block.BlockSize()])
	}
	return unpadPKCS7(out, block.BlockSize())
}

func parseRsaPublicKey(base64UrlDer string) (*rsa.PublicKey, error) {
	der, err := base64.RawURLEncoding.DecodeString(base64UrlDer)
	if err != nil {
		return nil, err
	}
	pubAny, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, err
	}
	pub, ok := pubAny.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("parsed public key is not RSA")
	}
	return pub, nil
}

func rsaEncryptBase64(plain string) (string, error) {
	pub, err := parseRsaPublicKey(defaultEncryptPublicKey)
	if err != nil {
		return "", err
	}
	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, pub, []byte(plain))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}
