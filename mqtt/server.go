package mqtt

import (
	"log"
	"net"

	"github.com/superscale/spire/config"
	"io"
)

// SessionHandler will be run in a goroutine for each connection the server accepts
type SessionHandler func(*Session)

// Server ...
type Server struct {
	bind        string
	listener    net.Listener
	sessHandler SessionHandler
}

// NewServer instantiates a new server that listens on the address passed in "bind"
func NewServer(bind string, sessHandler SessionHandler) *Server {
	if sessHandler == nil {
		return nil
	}

	return &Server{
		bind:        bind,
		sessHandler: sessHandler,
	}
}

// Run ...
func (s *Server) Run() {
	var err error
	if s.listener, err = net.Listen("tcp", s.bind); err != nil {
		if err != io.EOF {
			log.Println(err)
		}
		return
	}

	log.Println("listening on", s.bind)
	for {
		conn, err := s.listener.Accept()

		if err != nil {
			if err != io.EOF {
				log.Println(err)
			}
		} else {
			go s.sessHandler(NewSession(conn, config.Config.IdleConnectionTimeout))
		}
	}
}
