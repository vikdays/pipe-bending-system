package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"strings"
	"unicode/utf8"

	"changeme/internal/apperror"
	"golang.org/x/crypto/argon2"
)

const passwordPrefix = "$argon2id$v=19$m=65536,t=3,p=1$"

func validPassword(password string) bool {
	return utf8.ValidString(password) && utf8.RuneCountInString(password) >= 15 &&
		utf8.RuneCountInString(password) <= 128 && len(password) <= 512 && strings.TrimSpace(password) != ""
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, 3, 64*1024, 1, 32)
	return passwordPrefix + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(key), nil
}

func verifyPassword(password, encoded string) bool {
	if !strings.HasPrefix(encoded, passwordPrefix) || len(password) > 512 {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(encoded, passwordPrefix), "$")
	if len(parts) != 2 {
		return false
	}
	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[0])
	if err != nil || len(salt) != 16 {
		return false
	}
	expected, err := base64.RawStdEncoding.Strict().DecodeString(parts[1])
	if err != nil || len(expected) != 32 {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, 3, 64*1024, 1, 32)
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func passwordPolicyError() error {
	return apperror.New(apperror.ValidationError, "Пароль должен содержать от 15 до 128 символов. Используйте длинную уникальную фразу.")
}
