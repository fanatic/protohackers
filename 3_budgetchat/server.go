package budgetchat

import (
	"context"
	"errors"
	"log"
	"net"
	"regexp"
	"sync"

	proxyproto "github.com/pires/go-proxyproto"
)

type Server struct {
	Addr   string
	l      net.Listener
	cancel context.CancelFunc
	wg     sync.WaitGroup

	Room *Room
}

func NewServer(ctx context.Context, port string) (*Server, error) {
	ctx, cancel := context.WithCancel(ctx)

	var lc net.ListenConfig
	l, err := lc.Listen(ctx, "tcp", "0.0.0.0:"+port)
	if err != nil {
		cancel()
		return nil, err
	}

	// Wrap listener in a proxyproto listener
	l = &proxyproto.Listener{Listener: l}

	log.Printf("3_budgetchat at=server.listening addr=%q\n", l.Addr().String())
	s := &Server{Addr: l.Addr().String(), l: l, cancel: cancel, Room: NewRoom()}

	go s.acceptLoop(ctx)

	return s, nil
}

func (s *Server) Close() error {
	// Stop accepting new connections
	s.cancel()

	// Stop listening on port
	s.l.Close()

	// Wait for all connections to gracefully close (allow systemd to sigkill us)
	s.wg.Wait()
	return nil
}

func (s *Server) acceptLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := s.l.Accept()
			if errors.Is(err, net.ErrClosed) {
				return
			}
			if err != nil {
				log.Printf("3_budgetchat at=accept err=%q\n", err)
				continue
			}
			s.wg.Add(1)
			go func() {
				s.handleConn(conn)
				s.wg.Done()
			}()
		}
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	log.Printf("3_budgetchat at=handle-connection.start remote-addr=%q\n", conn.RemoteAddr())

	sess, err := NewSession(conn)
	if err != nil {
		log.Printf("3_budgetchat at=handle-connection.error remote-addr=%q err=%q\n", conn.RemoteAddr(), err.Error())
	}

	err = sess.SendMessage("Welcome to budgetchat! What shall I call you?")
	if err != nil {
		log.Printf("3_budgetchat at=handle-connection.username remote-addr=%q err=%q\n", conn.RemoteAddr(), err.Error())
	}
	if err := sess.Loop(s.Room); err != nil {
		sess.SendMessage(err.Error())
	}

	sess.Close()
	log.Printf("3_budgetchat at=handle-connection.finish remote-addr=%q\n", conn.RemoteAddr())
}

func isAlphaNumeric(word string) bool {
	return regexp.MustCompile(`^[a-zA-Z0-9]*$`).MatchString(word)
}
