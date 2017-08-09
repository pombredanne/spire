package service

import (
	"log"
	"net"
)

// ConnectionHandler will be run in a goroutine for each connection the server accepts
type ConnectionHandler func(net.Conn)

// Server ...
type Server struct {
	bind        string
	listener    net.Listener
	quit        chan bool
	connHandler ConnectionHandler
}

// NewServer instantiates a new server that listens on the address passed in "bind"
func NewServer(bind string, connHandler ConnectionHandler) *Server {
	if connHandler == nil {
		return nil
	}

	return &Server{
		bind:        bind,
		connHandler: connHandler,
		quit:        make(chan bool),
	}
}

// Shutdown stops the server from listening
func (s *Server) Shutdown() {
	if s.listener == nil {
		return
	}
	s.quit <- true
	s.listener.Close()
}

// Run ...
func (s *Server) Run() {
	var err error
	if s.listener, err = net.Listen("tcp", s.bind); err != nil {
		log.Println(err)
		return
	}

	log.Println("listening on", s.bind)
	for {
		conn, err := s.listener.Accept()

		if err != nil {
			select {
			case <-s.quit:
				return
			default:
			}

			log.Println(err)
			return
		}
		go s.connHandler(conn)
	}
}
