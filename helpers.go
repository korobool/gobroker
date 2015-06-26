package main

import (
	"bytes"
	"encoding/binary"
)

func identityIntToString(identity uint32) string {
	buf := new(bytes.Buffer)
	buf.WriteByte(0x0)
	_ = binary.Write(buf, binary.LittleEndian, identity)
	return string(buf.Bytes())
}
