package jwt

import "time"

// Options is a set of options that defines claims that are included in a token.
type Options struct {
	Audience       string                 // Audience is the "aud" claim.
	ExpirationTime time.Time              // ExpirationTime is the "exp" claim.
	Issuer         string                 // Issuer is the "iss" claim.
	JWTID          string                 // JWTID is the "jti" claim.
	NotBefore      time.Time              // NotBefore is the "nbf" claim.
	Subject        string                 // Subject is the "sub" claim.
	Timestamp      bool                   // Timestamp defines whether the JWT has the "iat" (issued at) claim set.
	KeyID          string                 // KeyID is the "kid" header claim.
	Public         map[string]interface{} // Public is a collection of public claims that are included to the JWT's payload.
}
