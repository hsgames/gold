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
	Read(conn net.Conn, readLimit int, withReadPool bool) ([]byte, error)
}

type Writer interface {
	Write(conn net.Conn, data []byte) (int, error)
}

func defaultReader() Reader {
	return &reader{}
}

func defaultWriter() Writer {
	return &writer{}
}

const defaultHeaderSize = 4

type reader struct{}

func (r *reader) Read(conn net.Conn, readLimit int, withReadPool bool) (data []byte, err error) {
	var header [defaultHeaderSize]byte

	if _, err = io.ReadFull(conn, header[:]); err != nil {
		err = fmt.Errorf("tcp: reader read header error [%w]", err)
		return
	}

	headerValue := int(binary.BigEndian.Uint32(header[:]))
	if headerValue > readLimit {
		err = fmt.Errorf("tcp: reader read packet size [%d] > [%d]", headerValue, readLimit)
		return
	}

	dataSize := headerValue - defaultHeaderSize
	if dataSize < 0 {
		err = fmt.Errorf("tcp: reader read data size [%d] < 0", dataSize)
		return
	}

	if withReadPool {
		data = bytespool.Get(dataSize)
	} else {
		data = make([]byte, dataSize)
	}

	if _, err = io.ReadFull(conn, data); err != nil {
		err = fmt.Errorf("tcp: reader read data error [%w]", err)

		if withReadPool {
			bytespool.Put(data)
		}
		data = nil

		return
	}

	return
}

type writer struct{}

func (w *writer) Write(conn net.Conn, data []byte) (n int, err error) {
	dataSize := len(data)

	headerValue := uint64(defaultHeaderSize) + uint64(dataSize)
	if headerValue > math.MaxUint32 {
		err = fmt.Errorf("tcp: writer write header value [%d] > math.MaxUint32", headerValue)
		return
	}

	packet := bytespool.Get(defaultHeaderSize + dataSize)
	defer bytespool.Put(packet)

	binary.BigEndian.PutUint32(packet, uint32(headerValue))
	copy(packet[defaultHeaderSize:], data)

	if n, err = conn.Write(packet); err != nil {
		err = fmt.Errorf("tcp: writer write packet error [%w]", err)
		return
	}

	return
}
