package sessions

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

// HashPassword is shared with operator/setup flows; no credentials are seeded.
func HashPassword(password string) (string, error) {
	if !utf8.ValidString(password) || utf8.RuneCountInString(password) < 12 || utf8.RuneCountInString(password) > 128 {
		return "", errors.New("password must contain 12 to 128 characters")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, 3, 65536, 1, 32)
	return fmt.Sprintf("$argon2id$v=19$m=65536,t=3,p=1$%s$%s", base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

// VerifyPassword deliberately accepts only the application's bounded hash profile.
// Invalid/missing hashes still perform the same Argon2id work as a valid hash.
func VerifyPassword(encoded, password string) bool {
	p := strings.Split(encoded, "$")
	salt, expected := make([]byte, 16), make([]byte, 32)
	valid := false
	if len(p) == 6 && p[1] == "argon2id" && p[2] == "v=19" && p[3] == "m=65536,t=3,p=1" {
		s, e1 := base64.RawStdEncoding.DecodeString(p[4])
		h, e2 := base64.RawStdEncoding.DecodeString(p[5])
		if e1 == nil && e2 == nil && len(s) == 16 && len(h) == 32 {
			salt, expected, valid = s, h, true
		}
	}
	actual := argon2.IDKey([]byte(password), salt, 3, 65536, 1, 32)
	return subtle.ConstantTimeCompare(actual, expected) == 1 && valid
}
