package net

import (
	"net"
)

type Conn interface {
	Shutdown()
	Close()
	Write(data []byte)
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	UserData() any
	SetUserData(data any)
	IsClosed() bool
	Done() chan struct{}
	String() string
}

type Handler interface {
	OnOpen(conn Conn) error
	OnClose(conn Conn) error
	OnRead(conn Conn, data []byte) error
	OnWrite(conn Conn, data []byte) error
}
