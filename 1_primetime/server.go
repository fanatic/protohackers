package primetime

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"math/big"
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

	log.Printf("1_primetime at=server.listening addr=%q\n", l.Addr().String())
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
				log.Printf("1_primetime at=accept err=%q\n", err)
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

	log.Printf("1_primetime at=handle-connection.start remote-addr=%q\n", conn.RemoteAddr())

	// Read through connection bytes line-by-line
	sc := bufio.NewScanner(conn)
	for sc.Scan() {
		var req Request
		if err := json.Unmarshal(sc.Bytes(), &req); err != nil {
			handleError(conn, err)
			continue
		}
		if err := handleRequest(conn, req); err != nil {
			handleError(conn, err)
			continue
		}
	}

	log.Printf("1_primetime at=handle-connection.finish remote-addr=%q\n", conn.RemoteAddr())
}

type Request struct {
	Method string  `json:"method"`
	Number float64 `json:"number"`
}

type Response struct {
	Method string `json:"method"`
	Prime  bool   `json:"prime"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func handleRequest(w io.Writer, req Request) error {
	if req.Method == "isPrime" {
		resp := handleIsPrime(req)
		return json.NewEncoder(w).Encode(&resp)
	} else {
		handleError(w, fmt.Errorf("unsupported method"))
	}
	return nil
}

func handleIsPrime(req Request) Response {
	log.Printf("1_primetime at=handle-request.start method=%q number=%f\n", req.Method, req.Number)
	defer log.Printf("1_primetime at=handle-request.finish method=%q number=%f\n", req.Method, req.Number)

	resp := Response{Method: req.Method, Prime: false}

	// Only whole numbers can be prime
	if req.Number == math.Trunc(req.Number) {
		z := big.NewInt(int64(req.Number))
		// n = 20 gives a false positive rate 0.000,000,000,001
		resp.Prime = z.ProbablyPrime(20)
	}

	return resp
}

func handleError(w io.Writer, e error) {
	resp := ErrorResponse{
		Error: e.Error(),
	}
	_ = json.NewEncoder(w).Encode(&resp)
}
