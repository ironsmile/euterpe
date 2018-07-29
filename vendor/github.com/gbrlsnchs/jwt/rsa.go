package jwt

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"errors"
)

var (
	ErrNoRSAPrivKey = errors.New("jwt.(Signer).Sign: RSA private key is nil")
	ErrNoRSAPubKey  = errors.New("jwt.(Signer).Verify: RSA public key is nil")
)

type rsasha struct {
	priv *rsa.PrivateKey
	pub  *rsa.PublicKey
	hash crypto.Hash
	alg  string
}

// RS256 creates a signing method using RSA and SHA-256.
func RS256(priv *rsa.PrivateKey, pub *rsa.PublicKey) Signer {
	return &rsasha{priv: priv, pub: pub, hash: crypto.SHA256, alg: MethodRS256}
}

// RS384 creates a signing method using RSA and SHA-384.
func RS384(priv *rsa.PrivateKey, pub *rsa.PublicKey) Signer {
	return &rsasha{priv: priv, pub: pub, hash: crypto.SHA384, alg: MethodRS384}
}

// RS512 creates a signing method using RSA and SHA-512.
func RS512(priv *rsa.PrivateKey, pub *rsa.PublicKey) Signer {
	return &rsasha{priv: priv, pub: pub, hash: crypto.SHA512, alg: MethodRS512}
}

func (r *rsasha) Sign(msg []byte) ([]byte, error) {
	if r.priv == nil {
		return nil, ErrNoRSAPrivKey
	}

	hh := r.hash.New()
	var err error

	if _, err = hh.Write(msg); err != nil {
		return nil, err
	}

	sig, err := rsa.SignPKCS1v15(rand.Reader, r.priv, r.hash, hh.Sum(nil))

	if err != nil {
		return nil, err
	}

	return sig, nil
}

func (r *rsasha) String() string {
	return r.alg
}

func (r *rsasha) Verify(msg, sig []byte) error {
	if r.pub == nil {
		return ErrNoRSAPubKey
	}

	hh := r.hash.New()
	var err error

	if _, err = hh.Write(msg); err != nil {
		return err
	}

	return rsa.VerifyPKCS1v15(r.pub, r.hash, hh.Sum(nil), sig)
}
