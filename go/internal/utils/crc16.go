package utils

import (
	"crypto/md5"
	"encoding/hex"
)

func GetChecksum(s string) string {
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}
