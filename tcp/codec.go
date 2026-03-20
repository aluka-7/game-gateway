package tcp

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/aluka-7/game-gateway/dto"
	pb "github.com/aluka-7/game-gateway/tcp/proto"
	"github.com/golang/protobuf/proto"
)

const (
	frameHeaderLen = 4
	maxFrameSize   = 4 * 1024 * 1024
)

func EncodeReq(msg *dto.CommonReq) ([]byte, error) {
	packet := &pb.TcpMessage{
		Server: msg.Server,
		Event:  msg.Event,
		Seq:    msg.Seq,
		UserId: msg.UserId,
		Data:   msg.Data,
	}
	body, err := proto.Marshal(packet)
	if err != nil {
		return nil, err
	}
	return EncodeFrame(body), nil
}

func DecodeRes(payload []byte, alias string) (*dto.CommonRes, error) {
	packet := new(pb.TcpMessage)
	if err := proto.Unmarshal(payload, packet); err != nil {
		return nil, err
	}
	res := &dto.CommonRes{
		Server: alias,
		Event:  packet.Event,
		Seq:    packet.Seq,
		UserId: packet.UserId,
		Code:   int(packet.Code),
		Msg:    packet.Msg,
		Data:   packet.Data,
	}
	return res, nil
}

func EncodeFrame(body []byte) []byte {
	frame := make([]byte, frameHeaderLen+len(body))
	binary.BigEndian.PutUint32(frame[:frameHeaderLen], uint32(len(body)))
	copy(frame[frameHeaderLen:], body)
	return frame
}

func ReadFrame(r io.Reader) ([]byte, error) {
	header := make([]byte, frameHeaderLen)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}
	n := binary.BigEndian.Uint32(header)
	if n == 0 {
		return nil, fmt.Errorf("invalid empty frame")
	}
	if n > maxFrameSize {
		return nil, fmt.Errorf("frame too large: %d", n)
	}
	payload := make([]byte, int(n))
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}
	return payload, nil
}
