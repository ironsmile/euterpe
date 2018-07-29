package jwt

type header struct {
	Algorithm string `json:"alg,omitempty"`
	KeyID     string `json:"kid,omitempty"`
	Type      string `json:"typ,omitempty"`
}
