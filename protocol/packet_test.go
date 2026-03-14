package protocol

import (
	"encoding/binary"
	"testing"
)

func TestPack(t *testing.T) {
	body := []byte("hello")
	msgID := uint16(42)

	packet := Pack(msgID, body)

	if got, want := len(packet), 4+2+len(body); got != want {
		t.Fatalf("unexpected packet size: got %d, want %d", got, want)
	}

	if got, want := binary.BigEndian.Uint32(packet[0:4]), uint32(2+len(body)); got != want {
		t.Fatalf("unexpected payload length: got %d, want %d", got, want)
	}

	if got := binary.BigEndian.Uint16(packet[4:6]); got != msgID {
		t.Fatalf("unexpected msg id: got %d, want %d", got, msgID)
	}

	if got := string(packet[6:]); got != string(body) {
		t.Fatalf("unexpected body: got %q, want %q", got, body)
	}
}
