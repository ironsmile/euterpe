package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"hash"
)

var (
	ErrNoHMACKey   = errors.New("jwt.(Signer).Sign: HMAC key is empty")
	ErrHMACInvalid = errors.New("jwt.(Signer).Verify: HMAC validation failed")
)

type hmacsha struct {
	key  []byte
	hash func() hash.Hash
	alg  string
}

// HS256 creates a signing method using HMAC and SHA-256.
func HS256(key string) Signer {
	return &hmacsha{key: []byte(key), hash: sha256.New, alg: MethodHS256}
}

// HS384 creates a signing method using HMAC and SHA-384.
func HS384(key string) Signer {
	return &hmacsha{key: []byte(key), hash: sha512.New384, alg: MethodHS384}
}

// HS512 creates a signing method using HMAC and SHA-512.
func HS512(key string) Signer {
	return &hmacsha{key: []byte(key), hash: sha512.New, alg: MethodHS512}
}

func (h *hmacsha) Sign(msg []byte) ([]byte, error) {
	if len(h.key) == 0 {
		return nil, ErrNoHMACKey
	}

	hh := hmac.New(h.hash, h.key)

	if _, err := hh.Write(msg); err != nil {
		return nil, err
	}

	return hh.Sum(nil), nil
}

func (h *hmacsha) String() string {
	return h.alg
}

func (h *hmacsha) Verify(msg, sig []byte) error {
	sig2, err := h.Sign(msg)

	if err != nil {
		return err
	}

	if !hmac.Equal(sig, sig2) {
		return ErrHMACInvalid
	}

	return nil
}
