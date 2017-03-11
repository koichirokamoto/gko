package gko

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"math/rand"
	"time"
)

var letters = []byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// RandSeq returns random n digit string.
func RandSeq(n int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// Sha512 returns sha512 hased string.
func Sha512(s string) string {
	return getHash(s, sha512.New())
}

// Sha384 returns sha384 hashed string.
func Sha384(s string) string {
	return getHash(s, sha512.New384())
}

// Sha256 returns sha256 hashed string.
func Sha256(s string) string {
	return getHash(s, sha256.New())
}

// Sha224 returns sha224 hashed string.
func Sha224(s string) string {
	return getHash(s, sha256.New224())
}

// Sha1 returns sha1 hashed string.
func Sha1(s string) string {
	return getHash(s, sha1.New())
}

// MD5 returns md5 hashed string.
func MD5(s string) string {
	return getHash(s, md5.New())
}

func getHash(s string, h hash.Hash) string {
	h.Write([]byte(s))
	sum := h.Sum(nil)
	return hex.EncodeToString(sum)
}
