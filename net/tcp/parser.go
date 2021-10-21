package tcp

import (
	"bufio"
	"encoding/binary"
	gbufio "github.com/hsgames/gold/bufio"
	"github.com/hsgames/gold/log"
	"github.com/pkg/errors"
	"math"
)

const packetParserHeaderSize = 2

type NewParserFunc = func(log.Logger) Parser

type Parser interface {
	ReadMessage(br *bufio.Reader, maxReadMsgSize int) ([]byte, error)
	WriteMessage(bw *bufio.Writer, data []byte, maxWriteMsgSize int) (int, error)
}

type PacketParser struct {
	header [packetParserHeaderSize]byte
	logger log.Logger
}

func DefaultParser(logger log.Logger) Parser {
	return &PacketParser{logger: logger}
}

func (p *PacketParser) ReadMessage(br *bufio.Reader, maxReadMsgSize int) ([]byte, error) {
	header, err := gbufio.PopReader(br, packetParserHeaderSize)
	if err != nil {
		return nil, err
	}
	msgSize := int(binary.BigEndian.Uint16(header)) - packetParserHeaderSize
	if msgSize <= 0 {
		return nil, errors.Errorf("tcp: parser read msg size %d <= 0", msgSize)
	}
	if msgSize > maxReadMsgSize {
		return nil, errors.Errorf("tcp: parser read msg size %d > %d",
			msgSize, maxReadMsgSize)
	}
	body, err := gbufio.PopReader(br, msgSize)
	if err != nil {
		return nil, err
	}
	data := make([]byte, msgSize)
	copy(data, body)
	return data, nil
}

func (p *PacketParser) WriteMessage(bw *bufio.Writer, data []byte, maxWriteMsgSize int) (int, error) {
	msgSize := len(data)
	if msgSize <= 0 {
		return 0, errors.Errorf("tcp: parser write msg size %d <= 0", msgSize)
	}
	if msgSize > maxWriteMsgSize {
		return 0, errors.Errorf("tcp: parser write msg size %d > %d",
			msgSize, maxWriteMsgSize)
	}
	headerValue := uint64(packetParserHeaderSize) + uint64(msgSize)
	if headerValue > math.MaxUint16 {
		return 0, errors.Errorf(
			"tcp: parser write header value %d > math.MaxUint16", headerValue)
	}
	header := p.header[:]
	binary.BigEndian.PutUint16(header, uint16(headerValue))
	nh, err := bw.Write(header)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	nd, err := bw.Write(data)
	if err != nil {
		return 0, errors.WithStack(err)
	}
	return nh + nd, nil
}
