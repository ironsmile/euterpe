package jwt

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"hash"
	"math/big"
)

var (
	ErrNoECDSAPrivKey = errors.New("jwt.(Signer).Sign: ECDSA private key is nil")
	ErrNoECDSAPubKey  = errors.New("jwt.(Signer).Sign: ECDSA public key is nil")
	ErrECSDAInvalid   = errors.New("jwt.(Signer).Verify: ECDSA validation failed")
	ErrECDSASigLen    = errors.New("jwt.(Signer).Verify: ECDSA signature has unexpected size")
)

type ecdsasha struct {
	priv *ecdsa.PrivateKey
	pub  *ecdsa.PublicKey
	hash func() hash.Hash
	alg  string
}

// ES256 creates a signing method using ECDSA and SHA-256.
func ES256(priv *ecdsa.PrivateKey, pub *ecdsa.PublicKey) Signer {
	return &ecdsasha{priv: priv, pub: pub, hash: sha256.New, alg: MethodES256}
}

// ES384 creates a signing method using ECDSA and SHA-384.
func ES384(priv *ecdsa.PrivateKey, pub *ecdsa.PublicKey) Signer {
	return &ecdsasha{priv: priv, pub: pub, hash: sha512.New384, alg: MethodES384}
}

// ES512 creates a signing method using ECDSA and SHA-512.
func ES512(priv *ecdsa.PrivateKey, pub *ecdsa.PublicKey) Signer {
	return &ecdsasha{priv: priv, pub: pub, hash: sha512.New, alg: MethodES512}
}

func (e *ecdsasha) Sign(msg []byte) ([]byte, error) {
	if e.priv == nil {
		return nil, ErrNoECDSAPrivKey
	}

	hh := e.hash()
	var err error

	if _, err = hh.Write(msg); err != nil {
		return nil, err
	}

	r, s, err := ecdsa.Sign(rand.Reader, e.priv, hh.Sum(nil))

	if err != nil {
		return nil, err
	}

	byteSize := e.byteSize(e.priv.Params().BitSize)
	rbytes := r.Bytes()
	rsig := make([]byte, byteSize)

	copy(rsig[byteSize-len(rbytes):], rbytes)

	sbytes := s.Bytes()
	ssig := make([]byte, byteSize)

	copy(ssig[byteSize-len(sbytes):], sbytes)

	return append(rsig, ssig...), nil
}

func (e *ecdsasha) String() string {
	return e.alg
}

func (e *ecdsasha) Verify(msg, sig []byte) error {
	if e.pub == nil {
		return ErrNoECDSAPubKey
	}

	byteSize := e.byteSize(e.pub.Params().BitSize)

	if len(sig) != byteSize*2 {
		return ErrECDSASigLen
	}

	r := big.NewInt(0).SetBytes(sig[:byteSize])
	s := big.NewInt(0).SetBytes(sig[byteSize:])
	hh := e.hash()

	if _, err := hh.Write(msg); err != nil {
		return err
	}

	if !ecdsa.Verify(e.pub, hh.Sum(nil), r, s) {
		return ErrECSDAInvalid
	}

	return nil
}

func (e *ecdsasha) byteSize(bitSize int) int {
	byteSize := bitSize / 8

	if bitSize%8 > 0 {
		return byteSize + 1
	}

	return byteSize
}
