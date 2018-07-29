package jwt

const (
	MethodHS256 = "HS256" // Method name for HMAC and SHA-256.
	MethodHS384 = "HS384" // Method name for HMAC and SHA-384.
	MethodHS512 = "HS512" // Method name for HMAC and SHA-512.
	MethodRS256 = "RS256" // Method name for RSA and SHA-256.
	MethodRS384 = "RS384" // Method name for RSA and SHA-384.
	MethodRS512 = "RS512" // Method name for RSA and SHA-512.
	MethodES256 = "ES256" // Method name for ECDSA and SHA-256.
	MethodES384 = "ES384" // Method name for ECDSA and SHA-384.
	MethodES512 = "ES512" // Method name for ECDSA and SHA-512.
	MethodNone  = "none"  // Method name for "none".
	algKey      = "alg"
	audKey      = "aud"
	expKey      = "exp"
	iatKey      = "iat"
	issKey      = "iss"
	jtiKey      = "jti"
	kidKey      = "kid"
	nbfKey      = "nbf"
	subKey      = "sub"
	typKey      = "typ"
)
