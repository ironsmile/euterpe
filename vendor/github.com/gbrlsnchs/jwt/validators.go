package jwt

import (
	"errors"
	"time"
)

var (
	ErrAlgorithmMismatch = errors.New("jwt: Algorithm mismatch")
	ErrAudienceMismatch  = errors.New("jwt: Audience mismatch")
	ErrTokenExpired      = errors.New("jwt: Token expired")
	ErrTokenFromFuture   = errors.New("jwt: Token issued at the future")
	ErrTokenTooYoung     = errors.New("jwt: Token is not valid yet")
	ErrIssuerMismatch    = errors.New("jwt: Issuer mismatch")
	ErrJWTIDMismatch     = errors.New("jwt: JWTID mismatch")
	ErrSubjectMismatch   = errors.New("jwt: Subject mismatch")
)

// AlgorithmValidator validates the "alg" claim.
func AlgorithmValidator(alg string) ValidatorFunc {
	return func(jot *JWT) error {
		if alg != jot.Algorithm() {
			return ErrAlgorithmMismatch
		}

		return nil
	}
}

// AudienceValidator validates the "aud" claim.
func AudienceValidator(aud string) ValidatorFunc {
	return func(jot *JWT) error {
		if jot.Audience() != aud {
			return ErrAudienceMismatch
		}

		return nil
	}
}

// ExpirationTimeValidator validates the "exp" claim.
func ExpirationTimeValidator(now time.Time) ValidatorFunc {
	return func(jot *JWT) error {
		if exp := jot.ExpirationTime(); !exp.IsZero() && now.After(exp) {
			return ErrTokenExpired
		}

		return nil
	}
}

// IssuedAtValidator validates the "iat" claim.
func IssuedAtValidator(now time.Time) ValidatorFunc {
	return func(jot *JWT) error {
		if now.Before(jot.IssuedAt()) {
			return ErrTokenFromFuture
		}

		return nil
	}
}

// IssuerValidator validates the "iss" claim.
func IssuerValidator(iss string) ValidatorFunc {
	return func(jot *JWT) error {
		if jot.Issuer() != iss {
			return ErrIssuerMismatch
		}

		return nil
	}
}

// IDValidator validates the "jti" claim.
func IDValidator(jti string) ValidatorFunc {
	return func(jot *JWT) error {
		if jot.ID() != jti {
			return ErrJWTIDMismatch
		}

		return nil
	}
}

// NotBeforeValidator validates the "nbf" claim.
func NotBeforeValidator(now time.Time) ValidatorFunc {
	return func(jot *JWT) error {
		if now.Before(jot.NotBefore()) {
			return ErrTokenTooYoung
		}

		return nil
	}
}

// SubjectValidator validates the "sub" claim.
func SubjectValidator(sub string) ValidatorFunc {
	return func(jot *JWT) error {
		if jot.Subject() != sub {
			return ErrSubjectMismatch
		}

		return nil
	}
}
