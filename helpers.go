package main

import (
	"bytes"
	"encoding/binary"
	// "fmt"
	// "github.com/mssola/user_agent"
	"strings"
)

const (
	PlatformAndroid = "Android"
	PlatformIPhone  = "iPhone"
	PlatformOther   = "Other"
)

func identityIntToString(identity uint32) string {
	buf := new(bytes.Buffer)
	buf.WriteByte(0x0)
	_ = binary.Write(buf, binary.LittleEndian, identity)
	return string(buf.Bytes())
}

func getPlatform(uaHeader string) string {

	if strings.Contains(strings.ToLower(uaHeader), "iphone") {
		return PlatformIPhone
	}

	if strings.Contains(strings.ToLower(uaHeader), "android") {
		return PlatformAndroid
	}

	return PlatformOther

}

func getDeviceType(platform string) string {
	if platform == PlatformOther {
		return "pc"
	}
	return "mobile"
}
