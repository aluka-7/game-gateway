package protocol

import (
	"bytes"
	"encoding/binary"
)

func Pack(msgId uint16, body []byte) []byte {

	length := 2 + len(body)

	buf := new(bytes.Buffer)

	binary.Write(buf, binary.BigEndian, uint32(length))
	binary.Write(buf, binary.BigEndian, msgId)

	buf.Write(body)

	return buf.Bytes()
}
