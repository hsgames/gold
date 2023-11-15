package tcp

import (
	"encoding/binary"
	"fmt"
	"github.com/hsgames/gold/pool/bytespool"
	"io"
	"math"
	"net"
)

type Reader interface {
	Read(conn net.Conn, maxReadMsgSize int) ([]byte, error)
}

type Writer interface {
	Write(conn net.Conn, data []byte, maxWriteMsgSize int) (int, error)
}

func defaultReader() Reader {
	return &reader{}
}

func defaultWriter() Writer {
	return &writer{}
}

const defaultHeaderSize = 2

type reader struct{}

func (r *reader) Read(conn net.Conn, maxReadMsgSize int) (data []byte, err error) {
	var header [defaultHeaderSize]byte

	if _, err = io.ReadFull(conn, header[:]); err != nil {
		return
	}

	msgSize := int(binary.BigEndian.Uint16(header[:])) - defaultHeaderSize
	if msgSize <= 0 {
		err = fmt.Errorf("tcp: reader read msg size [%d] <= 0", msgSize)
		return
	}

	if msgSize > maxReadMsgSize {
		err = fmt.Errorf("tcp: reader read msg size [%d] > [%d]", msgSize, maxReadMsgSize)
		return
	}

	data = make([]byte, msgSize)
	_, err = io.ReadFull(conn, data)

	return
}

type writer struct{}

func (w *writer) Write(conn net.Conn, data []byte, maxWriteMsgSize int) (n int, err error) {
	msgSize := len(data)
	if msgSize <= 0 {
		err = fmt.Errorf("tcp: writer write msg size [%d] <= 0", msgSize)
		return
	}
	if msgSize > maxWriteMsgSize {
		err = fmt.Errorf("tcp: writer write msg size [%d] > [%d]", msgSize, maxWriteMsgSize)
		return
	}

	headerValue := uint64(defaultHeaderSize) + uint64(msgSize)
	if headerValue > math.MaxUint16 {
		err = fmt.Errorf("tcp: writer write header value [%d] > math.MaxUint16", headerValue)
		return
	}

	buf := bytespool.Get(defaultHeaderSize + msgSize)
	defer bytespool.Put(buf)

	binary.BigEndian.PutUint16(buf, uint16(headerValue))
	copy(buf[defaultHeaderSize:], data)

	n, err = conn.Write(buf)

	return
}
