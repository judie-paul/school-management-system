// Package auth holds password/access-code hashing and random token helpers.
package auth

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

// Cost is the bcrypt cost. Tests lower it to keep them fast.
var Cost = bcrypt.DefaultCost

// HashSecret hashes a password or access code with bcrypt.
func HashSecret(secret string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(secret), Cost)
	return string(b), err
}

// CheckSecret reports whether secret matches hash.
func CheckSecret(hash, secret string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(secret)) == nil
}

var (
	dummyOnce sync.Once
	dummyHash []byte
)

// SpendCheckTime runs a bcrypt comparison against a throwaway hash so that an
// unknown username takes as long to reject as a wrong password.
func SpendCheckTime(secret string) {
	dummyOnce.Do(func() {
		dummyHash, _ = bcrypt.GenerateFromPassword([]byte("not-a-real-secret"), Cost)
	})
	bcrypt.CompareHashAndPassword(dummyHash, []byte(secret))
}

// RandomToken returns 32 random bytes, base64url encoded.
func RandomToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// 32 characters without look-alikes (0/O, 1/I), so byte%32 has no bias.
const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// NewAccessCode returns a random parent access code like "K7PQ-M2XD".
func NewAccessCode() string {
	b := make([]byte, 8)
	rand.Read(b)
	out := make([]byte, 0, 9)
	for i, c := range b {
		if i == 4 {
			out = append(out, '-')
		}
		out = append(out, codeAlphabet[c%32])
	}
	return string(out)
}

// NormalizeAccessCode uppercases a code and strips dashes and spaces, so
// "k7pq m2xd" and "K7PQ-M2XD" are equivalent. Codes are hashed in this form.
func NormalizeAccessCode(code string) string {
	return strings.Map(func(r rune) rune {
		if r == '-' || r == ' ' {
			return -1
		}
		return r
	}, strings.ToUpper(strings.TrimSpace(code)))
}
