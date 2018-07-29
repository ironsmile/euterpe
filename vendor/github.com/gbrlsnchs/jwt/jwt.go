package jwt

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

// ValidatorFunc is a function for running extra
// validators when parsing a JWT string.
type ValidatorFunc func(jot *JWT) error

var (
	// ErrEmptyAuthorization indicates that the "Authorization" header
	// doesn't have a token and, thus, extracting a token is impossible.
	ErrEmptyAuthorization = errors.New("jwt: no token could be extracted from header")
	// ErrMalformedToken indicates a token doesn't have
	// a valid format, as per the RFC 7519, section 7.2.
	ErrMalformedToken = errors.New("jwt: malformed token")
	// ErrNilCtxKey indicates that no context key is set for retrieving
	// JWTs from context objects. This error is resolved if a key is set.
	ErrNilCtxKey = errors.New("jwt: JWT context key is a nil value")
	// ErrCtxAssertion indicates a JWT could not be extracted from a context object
	// because the value it holds can not be asserted to a JWT pointer.
	ErrCtxAssertion = errors.New("jwt: unable to assert context value into JWT pointer")
)

// JWT is a JSON Web Token.
type JWT struct {
	header *header
	claims *claims
	raw    string
	sep    int
}

// FromContext extracts a JWT object from a given context.
func FromContext(ctx context.Context, key interface{}) (*JWT, error) {
	if key == nil {
		return nil, ErrNilCtxKey
	}

	v := ctx.Value(key)

	if token, ok := v.(string); ok {
		return FromString(token)
	}

	jot, ok := v.(*JWT)

	if !ok {
		return nil, ErrCtxAssertion
	}

	return jot, nil
}

// FromCookie extracts a JWT object from a given cookie.
func FromCookie(c *http.Cookie) (*JWT, error) {
	return FromString(c.Value)
}

// FromRequest builds a JWT from the token contained
// in the "Authorization" header.
func FromRequest(r *http.Request) (*JWT, error) {
	auth := r.Header.Get("Authorization")
	i := strings.IndexByte(auth, ' ')

	if i < 0 {
		return nil, ErrEmptyAuthorization
	}

	return FromString(auth[i+1:])
}

// FromString builds a JWT from a string representation
// of a JSON Web Token.
func FromString(s string) (*JWT, error) {
	sep1 := strings.IndexByte(s, '.')

	if sep1 < 0 {
		return nil, ErrMalformedToken
	}

	sep2 := strings.IndexByte(s[sep1+1:], '.')

	if sep2 < 0 {
		return nil, ErrMalformedToken
	}

	sep2 += sep1 + 1
	jot := &JWT{raw: s, sep: sep2}

	if err := jot.build(); err != nil {
		return nil, err
	}

	return jot, nil
}

// Algorithm returns the "alg" claim
// from the JWT's header.
func (j *JWT) Algorithm() string {
	return j.header.Algorithm
}

// Audience returns the "aud" claim
// from the JWT's payload.
func (j *JWT) Audience() string {
	return j.claims.aud
}

// Bytes returns a representation of the JWT
// as an array of bytes.
func (j *JWT) Bytes() []byte {
	return []byte(j.raw)
}

// ExpirationTime returns the "exp" claim
// from the JWT's payload.
func (j *JWT) ExpirationTime() time.Time {
	return j.claims.exp
}

// IssuedAt returns the "iat" claim
// from the JWT's payload.
func (j *JWT) IssuedAt() time.Time {
	return j.claims.iat
}

// Issuer returns the "iss" claim
// from the JWT's payload.
func (j *JWT) Issuer() string {
	return j.claims.iss
}

// ID returns the "jti" claim
// from the JWT's payload.
func (j *JWT) ID() string {
	return j.claims.jti
}

// KeyID returns the "kid" claim
// from the JWT's header.
func (j *JWT) KeyID() string {
	return j.header.KeyID
}

// NotBefore returns the "nbf" claim
// from the JWT's payload.
func (j *JWT) NotBefore() time.Time {
	return j.claims.nbf
}

// Public returns all public claims set.
func (j *JWT) Public() map[string]interface{} {
	return j.claims.pub
}

// Subject returns the "sub" claim
// from the JWT's payload.
func (j *JWT) Subject() string {
	return j.claims.sub
}

func (j *JWT) String() string {
	return j.raw
}

// Validate iterates over custom validator functions to validate the JWT.
func (j *JWT) Validate(vfuncs ...ValidatorFunc) error {
	for _, vfunc := range vfuncs {
		if err := vfunc(j); err != nil {
			return err
		}
	}

	return nil
}

// Verify verifies the Token's signature.
func (j *JWT) Verify(s Signer) error {
	var (
		sig []byte
		err error
	)

	if sig, err = decode(j.raw[j.sep+1:]); err != nil {
		return err
	}

	return s.Verify([]byte(j.raw[:j.sep]), sig)
}

func (j *JWT) build() error {
	var (
		p1, p2 = j.parts()
		dec    []byte
		err    error
	)

	if dec, err = decode(p1); err != nil {
		return err
	}

	if err = json.Unmarshal(dec, &j.header); err != nil {
		return err
	}

	if dec, err = decode(p2); err != nil {
		return err
	}

	if err = json.Unmarshal(dec, &j.claims); err != nil {
		return err
	}

	return nil
}

func (j *JWT) parts() (string, string) {
	sep := strings.IndexByte(j.raw[:j.sep], '.')

	return j.raw[:sep], j.raw[sep+1 : j.sep]
}
