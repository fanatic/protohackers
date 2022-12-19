package mobinthemiddle

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net"
	"sync"

	"github.com/dlclark/regexp2"
	proxyproto "github.com/pires/go-proxyproto"
)

type Server struct {
	Addr   string
	l      net.Listener
	cancel context.CancelFunc
	wg     sync.WaitGroup

	Boguscoin *regexp2.Regexp
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

	log.Printf("5_mobinthemiddle at=server.listening addr=%q\n", l.Addr().String())
	s := &Server{
		Addr:      l.Addr().String(),
		l:         l,
		cancel:    cancel,
		Boguscoin: regexp2.MustCompile(`(?<=^|\s)(7[a-zA-Z0-9]{25,34})(?=\s|$)`, regexp2.None),
	}

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
				log.Printf("5_mobinthemiddle at=accept err=%q\n", err)
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
	log.Printf("5_mobinthemiddle at=handle-connection.start remote-addr=%q\n", conn.RemoteAddr())

	client, err := net.Dial("tcp", "chat.protohackers.com:16963")
	if err != nil {
		log.Printf("5_mobinthemiddle at=client.err remote-addr=%q err=%s\n", conn.RemoteAddr(), err)
		return
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		s.oneWayCopy(conn, client)
		client.Close()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		s.oneWayCopy(client, conn)
		client.Close()
	}()

	wg.Wait()

	log.Printf("5_mobinthemiddle at=handle-connection.finish remote-addr=%q\n", conn.RemoteAddr())
}

func (s *Server) oneWayCopy(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)
	scanner.Split(ScanTerminatedLines)
	// optionally, resize scanner's capacity for lines over 64K, see next example
	for scanner.Scan() {
		t := scanner.Text()
		//log.Printf("5_mobinthemiddle at=copy msg=%q\n", t)

		b, _ := s.Boguscoin.Replace(t, "7YWHMfk9JZe0LM0g1ZauHuiSxhI", -1, -1)
		_, err := out.Write([]byte(b + "\n"))
		if errors.Is(err, net.ErrClosed) {
			return
		} else if err != nil {
			log.Printf("5_mobinthemiddle at=write.err err=%s\n", err)
			return
		}
	}

	err := scanner.Err()
	if errors.Is(err, net.ErrClosed) {
		return
	} else if err != nil {
		log.Printf("5_mobinthemiddle at=scan.err err=%s\n", err)
		return
	}
}

func ScanTerminatedLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		// We have a full newline-terminated line.
		return i + 1, dropCR(data[0:i]), nil
	}
	// Request more data.
	return 0, nil, nil
}

func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}
