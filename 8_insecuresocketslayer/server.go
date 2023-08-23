package insecuresocketslayer

import (
	"bufio"
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

	log.Printf("8_insecuresocketslayer at=server.listening addr=%q\n", l.Addr().String())
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
				log.Printf("8_insecuresocketslayer at=accept err=%q\n", err)
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
	log.Printf("8_insecuresocketslayer at=handle-connection.start remote-addr=%q\n", conn.RemoteAddr())

	cipherSpec, err := s.handleCipherSpec(conn)
	if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
		log.Printf("8_insecuresocketslayer at=handle-connection.finish remote-addr=%q\n", conn.RemoteAddr())
		return
	} else if err != nil {
		log.Printf("8_insecuresocketslayer at=handle-connection.err err=%s\n", err)
		return
	}

	crw := NewCipherReaderWriter(conn, cipherSpec)

	// Validate cipherSpec does not leave every byte of input unchanged (e.g. a no-op cipher)
	// This is a very naive check, but it's good enough for this challenge
	if err := crw.Validate(); err != nil {
		log.Printf("8_insecuresocketslayer at=handle-connection.err err=%s\n", err)
		return
	}

	err = s.handleApplication(crw, cipherSpec)
	if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
		log.Printf("8_insecuresocketslayer at=handle-connection.finish remote-addr=%q\n", conn.RemoteAddr())
		return
	} else if err != nil {
		log.Printf("8_insecuresocketslayer at=handle-connection.err err=%s\n", err)
		return
	}

	log.Printf("8_insecuresocketslayer at=handle-connection.finish remote-addr=%q\n", conn.RemoteAddr())
}

// The cipher spec is represented as a series of operations, with the operation types encoded by a single byte (and for 02 and 04, another byte encodes the operand), ending with a 00 byte, as follows:
func (s *Server) handleCipherSpec(conn net.Conn) ([]byte, error) {
	log.Printf("8_insecuresocketslayer at=handle-cipher-spec.start remote-addr=%q\n", conn.RemoteAddr())

	// Read the cipher spec ending with a 00 byte
	var cipherSpec []byte
	for {
		b := make([]byte, 1)
		_, err := conn.Read(b)
		if err != nil {
			return nil, err
		}
		if b[0] == 0 {
			break
		}
		cipherSpec = append(cipherSpec, b[0])
	}

	log.Printf("8_insecuresocketslayer at=handle-cipher-spec.finish spec=%x remote-addr=%q\n", cipherSpec, conn.RemoteAddr())
	return cipherSpec, nil
}

func (s *Server) handleApplication(conn io.ReadWriter, cipherSpec []byte) error {
	// Read the application data separated by newlines
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		msg := scanner.Text()

		log.Printf("<-- %s\n", string(msg))
		reply := []byte(findMaxToy(msg))
		log.Printf("--> %s\n", string(reply))

		// Send the message back to the client
		_, err := conn.Write(append(reply, "\n"...))
		if err != nil {
			return err
		}
	}

	return nil
}
