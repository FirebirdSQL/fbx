package wirecrypt

import (
	"crypto/rc4"
	"crypto/sha256"

	"github.com/nakagami/chacha20"
)

// Codec provides symmetric encrypt/decrypt for wire data.
type Codec interface {
	// Encrypt writes encrypted bytes of src into dst.
	Encrypt(dst, src []byte)
	// Decrypt writes decrypted bytes of src into dst.
	Decrypt(dst, src []byte)
}

// NewSymmetricCodec constructs a Codec for a symmetric wire plugin.
// Supported plugins: "Arc4", "ChaCha", "ChaCha64".
func NewSymmetricCodec(plugin string, sessionKey, nonce []byte) (Codec, error) {
	switch plugin {
	case "Arc4":
		return newRC4Codec(sessionKey)
	case "ChaCha", "ChaCha64":
		return newChaChaCodec(sessionKey, nonce)
	default:
		return nil, ErrNoCommonPlugin
	}
}

type rc4Codec struct {
	r *rc4.Cipher
	w *rc4.Cipher
}

func newRC4Codec(key []byte) (Codec, error) {
	r, err := rc4.NewCipher(key)
	if err != nil {
		return nil, err
	}
	w, err := rc4.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return &rc4Codec{r: r, w: w}, nil
}

func (c *rc4Codec) Encrypt(dst, src []byte) { c.w.XORKeyStream(dst, src) }
func (c *rc4Codec) Decrypt(dst, src []byte) { c.r.XORKeyStream(dst, src) }

type chachaCodec struct {
	r *chacha20.Cipher
	w *chacha20.Cipher
}

func newChaChaCodec(sessionKey, nonce []byte) (Codec, error) {
	// Derive 32-byte key via SHA-256, matching previous behavior.
	digest := sha256.New()
	digest.Write(sessionKey)
	key := digest.Sum(nil)
	r, err := chacha20.NewCipher(key, nonce, 0)
	if err != nil {
		return nil, err
	}
	w, err := chacha20.NewCipher(key, nonce, 0)
	if err != nil {
		return nil, err
	}
	return &chachaCodec{r: r, w: w}, nil
}

func (c *chachaCodec) Encrypt(dst, src []byte) { c.w.XORKeyStream(dst, src) }
func (c *chachaCodec) Decrypt(dst, src []byte) { c.r.XORKeyStream(dst, src) }
