package net

import "github.com/hsgames/gold/log"

type NewHandlerFunc = func(log.Logger) Handler

type Conn interface {
	Shutdown()
	Close()
	Write(data []byte)
	LocalAddr() string
	RemoteAddr() string
	EndPoint() EndPoint
	UserData() interface{}
	SetUserData(data interface{})
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
