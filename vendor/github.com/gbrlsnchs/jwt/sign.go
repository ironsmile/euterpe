package jwt

import (
	"encoding/json"
	"errors"
	"time"
)

var ErrNoSigner = errors.New("jwt.Sign: signer is nil")

// Sign builds a full JWT and signs its last part.
func Sign(s Signer, opts *Options) (string, error) {
	now := time.Now()

	if s == nil {
		return "", ErrNoSigner
	}

	if opts == nil {
		opts = &Options{}
	}

	jot := &JWT{
		header: &header{
			Algorithm: s.String(),
			KeyID:     opts.KeyID,
			Type:      "JWT",
		},
		claims: &claims{
			aud: opts.Audience,
			exp: opts.ExpirationTime,
			iss: opts.Issuer,
			jti: opts.JWTID,
			nbf: opts.NotBefore,
			sub: opts.Subject,
			pub: make(map[string]interface{}),
		},
	}

	for k, v := range opts.Public {
		jot.claims.pub[k] = v
	}

	if opts.Timestamp {
		jot.claims.iat = now
	}

	var token []byte
	p, err := json.Marshal(jot.header)

	if err != nil {
		return "", err
	}

	token = append(token, encode(p)...)

	p, err = json.Marshal(jot.claims)

	if err != nil {
		return "", err
	}

	token = append(token, '.')
	token = append(token, encode(p)...)

	p, err = s.Sign(token)

	if err != nil {
		return "", err
	}

	token = append(token, '.')
	token = append(token, encode(p)...)

	return string(token), nil
}
