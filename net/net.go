package net

import (
	"github.com/hsgames/gold/log"
	"net"
)

type NewHandlerFunc = func(log.Logger) Handler

type Conn interface {
	Shutdown()
	Close()
	Write(data []byte)
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	EndPoint() EndPoint
	UserData() any
	SetUserData(data any)
	IsClosed() bool
}

type Handler interface {
	OnOpen(conn Conn) error
	OnClose(conn Conn) error
	OnMessage(conn Conn, data []byte) error
}

type EndPoint interface {
	Name() string
	Addr() string
}
