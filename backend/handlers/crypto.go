package handlers

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"log"
	"os"
)

// getEncryptionKey derives a 32-byte AES-256 key from ENCRYPTION_KEY env var.
// Falls back to JWT_SECRET only in dev (no DATABASE_URL). Fatals in production if neither is set.
func getEncryptionKey() []byte {
	key := os.Getenv("ENCRYPTION_KEY")
	if key == "" {
		if os.Getenv("DATABASE_URL") != "" {
			// Produção: ENCRYPTION_KEY deve ser configurada separadamente do JWT_SECRET
			// para evitar que um vazamento do JWT comprometa as credenciais Oracle.
			if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
				log.Println("SECURITY WARNING: ENCRYPTION_KEY não configurada em produção — credenciais Oracle estão usando JWT_SECRET como chave de criptografia. Configure ENCRYPTION_KEY imediatamente.")
				key = jwtSecret
			} else {
				log.Fatal("FATAL: ENCRYPTION_KEY e JWT_SECRET não configuradas em produção — impossível criptografar credenciais Oracle.")
			}
		} else {
			key = "super-secret-key-change-me-in-prod"
		}
	}
	h := sha256.Sum256([]byte(key))
	return h[:]
}

// ValidateEncryptionKey emite aviso se ENCRYPTION_KEY não estiver configurada em produção.
// Deve ser chamada no startup (main.go) para garantir visibilidade nos logs.
func ValidateEncryptionKey() {
	if os.Getenv("ENCRYPTION_KEY") == "" && os.Getenv("DATABASE_URL") != "" {
		if os.Getenv("JWT_SECRET") != "" {
			log.Println("SECURITY WARNING: ENCRYPTION_KEY não configurada em produção — credenciais Oracle estão usando JWT_SECRET como chave de criptografia. Configure ENCRYPTION_KEY imediatamente.")
		} else {
			log.Fatal("FATAL: ENCRYPTION_KEY e JWT_SECRET não configuradas em produção.")
		}
	}
}

// EncryptField encrypts plaintext using AES-256-GCM.
// Returns a base64-encoded string containing nonce + ciphertext.
func EncryptField(plaintext string) (string, error) {
	key := getEncryptionKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptField decrypts a base64-encoded AES-256-GCM ciphertext produced by EncryptField.
func DecryptField(encoded string) (string, error) {
	key := getEncryptionKey()
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, ct := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// DecryptFieldWithFallback decrypts a field, returning the raw value on error.
// Used during migration: old plaintext values are returned as-is until re-saved.
func DecryptFieldWithFallback(encoded string) string {
	decrypted, err := DecryptField(encoded)
	if err != nil {
		return encoded // probably plaintext (not yet migrated)
	}
	return decrypted
}
