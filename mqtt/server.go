package mqtt

import (
	"log"
	"net"

	"github.com/superscale/spire/config"
)

// ConnectionHandler will be run in a goroutine for each connection the server accepts
type ConnectionHandler func(*Conn)

// Server ...
type Server struct {
	bind        string
	listener    net.Listener
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
	}
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
			log.Println(err)
		} else {
			go s.connHandler(NewConn(conn, config.Config.IdleConnectionTimeout))
		}
	}
}
