package main

import "net"

type User struct {
	Name string
	Addr string
	Ch   chan string
	conn net.Conn
}

func NewUser(conn net.Conn) *User {
	userAddr := conn.RemoteAddr().String()

	user := &User{
		Name: userAddr,
		Addr: userAddr,
		Ch:   make(chan string),
		conn: conn,
	}

	go user.handleMessage()
	return user
}

func (u *User) handleMessage() {

	for {
		msg := <-u.Ch
		u.conn.Write([]byte(msg + "\n"))
	}
}
