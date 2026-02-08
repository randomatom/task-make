package utils

import (
	"fmt"
	"hash/adler32"
)

func GetChecksum(s string) string {
	h := adler32.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%08x", h.Sum32())
}
