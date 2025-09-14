package main

import (
	"fmt"
	"io"
	"net"
	"sync"
)

type Server struct {
	Ip       string
	Port     int
	ch       chan string
	sessions map[string]*User
	lock     sync.RWMutex
}

func NewServer(ip string, port int) *Server {
	return &Server{
		Ip:       ip,
		Port:     port,
		ch:       make(chan string),
		sessions: make(map[string]*User),
	}
}

func (s *Server) broadCast(user *User, msg string) {
	broadMsg := "[" + user.Addr + "]" + user.Name + ":" + msg
	s.ch <- broadMsg
}

func (s *Server) handleBroadMsg() {
	for {
		msg := <-s.ch
		s.lock.RLock()
		for _, user := range s.sessions {
			user.Ch <- msg
		}
		s.lock.RUnlock()
	}
}

func (s *Server) handler(conn net.Conn) {

	user := NewUser(conn)
	s.lock.Lock()
	s.sessions[user.Name] = user
	s.lock.Unlock()

	s.broadCast(user, "上线")

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n == 0 {
				s.broadCast(user, "下线")
				return
			}

			if err != nil && err != io.EOF {
				fmt.Println("Conn Read err:", err)
			}

			msg := string(buf[:n-1])
			if msg == "who" {
				s.lock.RLock()
				for _, user := range s.sessions {
					msg := "[" + user.Addr + "]" + user.Name + ":" + "在线..."
					conn.Write([]byte(msg + "\n"))
				}
				s.lock.Lock()
			} else {
				s.broadCast(user, msg)
			}
		}
	}()

	select {}
}

func (s *Server) Start() {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.Ip, s.Port))
	if err != nil {
		fmt.Println("net.Listen err:", err)
		return
	}

	defer listener.Close()

	go s.handleBroadMsg()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("listener.Accept err:", err)
			continue
		}

		go s.handler(conn)
	}
}
