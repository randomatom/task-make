package utils

import (
	"encoding/binary"
	"hash/adler32"
)

func GetChecksum(s string) string {
	h := adler32.New()
	h.Write([]byte(s))
	return string(binary.BigEndian.AppendUint32(nil, h.Sum32()))
}
