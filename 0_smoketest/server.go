package smoketest

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"sync"

	proxyproto "github.com/pires/go-proxyproto"
)

type Server struct {
	Addr   string
	l      net.Listener
	cancel context.CancelFunc
	wg     sync.WaitGroup
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

	log.Printf("0_smoketest at=server.listening addr=%q\n", l.Addr().String())
	s := &Server{Addr: l.Addr().String(), l: l, cancel: cancel}

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
				log.Printf("0_smoketest at=accept err=%q\n", err)
				continue
			}
			s.wg.Add(1)
			go func() {
				handleConn(conn)
				s.wg.Done()
			}()
		}
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	//log.Printf("0_smoketest at=handle-connection remote-addr=%q\n", conn.RemoteAddr())
	_, err := io.Copy(conn, conn)
	if err == io.EOF {
		return
	} else if err != nil {
		log.Printf("0_smoketest at=handle-connection err=%q\n", err)
	}
	//log.Printf("0_smoketest at=handle-connection bytes=%d\n", n)
}
