package main

import (
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
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

	isLive := make(chan string)

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
				s.lock.RUnlock()
			} else if len(msg) > 7 && msg[:7] == "rename|" {
				newName := strings.Split(msg, "|")[1]
				s.lock.Lock()
				if _, ok := s.sessions[newName]; ok {
					conn.Write([]byte("用户名重复\n"))
				} else {
					delete(s.sessions, user.Name)
					s.sessions[newName] = user
					user.Name = newName
				}
				s.lock.Unlock()
			} else if len(msg) > 3 && msg[:3] == "to|" {
				toUser := strings.Split(msg, "|")[1]
				message := strings.Split(msg, "|")[2]
				s.lock.RLock()
				if u, ok := s.sessions[toUser]; ok {
					u.conn.Write([]byte(message + "\n"))
				} else {
					conn.Write([]byte("用户不存在\n"))
				}
				s.lock.RUnlock()
			} else {
				s.broadCast(user, msg)
			}

			isLive <- "0"
		}
	}()

	for {
		select {
		case <-isLive:
		case <-time.After(time.Second * 10):
			user.conn.Write([]byte("你已被踢下线\n"))
			delete(s.sessions, user.Name)
			close(user.Ch)
			conn.Close()
			return
		}
	}
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
