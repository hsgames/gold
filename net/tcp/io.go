package tcp

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"

	"github.com/hsgames/gold/pool/bytespool"
)

type Reader interface {
	Read(conn net.Conn, maxReadDataSize int, getReadData func(size int) []byte) ([]byte, error)
}

type Writer interface {
	Write(conn net.Conn, data []byte, maxWriteDataSize int) (int, error)
}

func defaultReader() Reader {
	return &reader{}
}

func defaultWriter() Writer {
	return &writer{}
}

const defaultHeaderSize = 2

type reader struct{}

func (r *reader) Read(conn net.Conn, maxReadDataSize int, getReadData func(size int) []byte) (data []byte, err error) {
	var header [defaultHeaderSize]byte

	if _, err = io.ReadFull(conn, header[:]); err != nil {
		return
	}

	dataSize := int(binary.BigEndian.Uint16(header[:])) - defaultHeaderSize
	if dataSize <= 0 {
		err = fmt.Errorf("tcp: reader read data size [%d] <= 0", dataSize)
		return
	}

	if dataSize > maxReadDataSize {
		err = fmt.Errorf("tcp: reader read data size [%d] > [%d]", dataSize, maxReadDataSize)
		return
	}

	data = getReadData(dataSize)

	_, err = io.ReadFull(conn, data)

	return
}

func (r *reader) Release(data []byte) error {
	bytespool.Put(data)
	return nil
}

type writer struct{}

func (w *writer) Write(conn net.Conn, data []byte, maxWriteDataSize int) (n int, err error) {
	dataSize := len(data)
	if dataSize <= 0 {
		err = fmt.Errorf("tcp: writer write data size [%d] <= 0", dataSize)
		return
	}
	if dataSize > maxWriteDataSize {
		err = fmt.Errorf("tcp: writer write data size [%d] > [%d]", dataSize, maxWriteDataSize)
		return
	}

	headerValue := uint64(defaultHeaderSize) + uint64(dataSize)
	if headerValue > math.MaxUint16 {
		err = fmt.Errorf("tcp: writer write header value [%d] > math.MaxUint16", headerValue)
		return
	}

	packet := bytespool.Get(defaultHeaderSize + dataSize)
	defer bytespool.Put(packet)

	binary.BigEndian.PutUint16(packet, uint16(headerValue))
	copy(packet[defaultHeaderSize:], data)

	n, err = conn.Write(packet)

	return
}
