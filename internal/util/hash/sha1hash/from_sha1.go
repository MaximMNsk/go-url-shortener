package sha1hash

import (
	"crypto/sha1"
	"encoding/hex"
)

func Create(input string, len int) string {
	h := sha1.New()

	h.Write([]byte(input))
	sha1Hash := hex.EncodeToString(h.Sum(nil))

	return sha1Hash[:len]
}
