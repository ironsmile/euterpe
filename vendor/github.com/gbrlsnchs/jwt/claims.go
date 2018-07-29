package jwt

import (
	"encoding/json"
	"time"
)

type claims struct {
	aud string
	exp time.Time
	iat time.Time
	iss string
	jti string
	nbf time.Time
	sub string
	pub map[string]interface{}
}

func (c *claims) MarshalJSON() ([]byte, error) {
	if c.pub == nil {
		c.pub = make(map[string]interface{})
	}

	if len(c.aud) > 0 {
		c.pub[audKey] = c.aud
	}

	if !c.exp.IsZero() {
		c.pub[expKey] = c.exp.Unix()
	}

	if !c.iat.IsZero() {
		c.pub[iatKey] = c.iat.Unix()
	}

	if len(c.iss) > 0 {
		c.pub[issKey] = c.iss
	}

	if len(c.jti) > 0 {
		c.pub[jtiKey] = c.jti
	}

	if !c.nbf.IsZero() {
		c.pub[nbfKey] = c.nbf.Unix()
	}

	if len(c.sub) > 0 {
		c.pub[subKey] = c.sub
	}

	return json.Marshal(c.pub)
}

func (c *claims) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &c.pub); err != nil {
		return err
	}

	if v, ok := c.pub[audKey].(string); ok {
		c.aud = v
	}

	delete(c.pub, audKey)

	if v, ok := c.pub[expKey].(float64); ok {
		c.exp = time.Unix(int64(v), 0)
	}

	delete(c.pub, expKey)

	if v, ok := c.pub[iatKey].(float64); ok {
		c.iat = time.Unix(int64(v), 0)
	}

	delete(c.pub, iatKey)

	if v, ok := c.pub[issKey].(string); ok {
		c.iss = v
	}

	delete(c.pub, issKey)

	if v, ok := c.pub[jtiKey].(string); ok {
		c.jti = v
	}

	delete(c.pub, jtiKey)

	if v, ok := c.pub[nbfKey].(float64); ok {
		c.nbf = time.Unix(int64(v), 0)
	}

	delete(c.pub, nbfKey)

	if v, ok := c.pub[subKey].(string); ok {
		c.sub = v
	}

	delete(c.pub, subKey)

	return nil
}
