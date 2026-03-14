package protocol

import (
	"encoding/binary"
)

func Pack(msgId uint16, body []byte) []byte {
	length := 2 + len(body)
	packet := make([]byte, 4+length)
	binary.BigEndian.PutUint32(packet[0:4], uint32(length))
	binary.BigEndian.PutUint16(packet[4:6], msgId)
	copy(packet[6:], body)
	return packet
}
