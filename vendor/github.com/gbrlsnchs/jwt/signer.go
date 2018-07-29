package jwt

type Signer interface {
	Sign(msg []byte) ([]byte, error)
	String() string
	Verify(msg, sig []byte) error
}
