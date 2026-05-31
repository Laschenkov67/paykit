package signing

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"hash"
)

func HMACSHA256Hex(key, message []byte) string {
	return hexHMAC(sha256.New, key, message)
}

func HMACSHA512Hex(key, message []byte) string {
	return hexHMAC(sha512.New, key, message)
}

func hexHMAC(h func() hash.Hash, key, message []byte) string {
	mac := hmac.New(h, key)
	mac.Write(message)
	return hex.EncodeToString(mac.Sum(nil))
}

func EqualHex(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
