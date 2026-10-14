package api_key

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
)

const secretSize = 32

func GenerateSecret() (encodedSecret string, keyHash []byte, err error) {
	secret := make([]byte, secretSize)
	if _, err := rand.Read(secret); err != nil {
		return "", nil, err
	}

	sum := sha256.Sum256(secret)

	return base64.RawURLEncoding.EncodeToString(secret), append([]byte(nil), sum[:]...), nil
}

func VerifySecret(encodedSecret string, storedHash []byte) bool {
	secret, err := base64.RawURLEncoding.DecodeString(encodedSecret)
	if err != nil || len(secret) != secretSize ||
		len(storedHash) != sha256.Size {
		return false
	}

	sum := sha256.Sum256(secret)
	return subtle.ConstantTimeCompare(sum[:], storedHash) == 1
}
