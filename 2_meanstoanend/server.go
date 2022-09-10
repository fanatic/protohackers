package meanstoanend

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	proxyproto "github.com/pires/go-proxyproto"
)

type Server struct {
	Addr   string
	l      net.Listener
	cancel context.CancelFunc
	wg     sync.WaitGroup

	sync.Mutex
	timestamps []int32
	values     map[int32]int32
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

	log.Printf("2_meanstoanend at=server.listening addr=%q\n", l.Addr().String())
	s := &Server{Addr: l.Addr().String(), l: l, cancel: cancel, values: map[int32]int32{}}

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
				log.Printf("2_meanstoanend at=accept err=%q\n", err)
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

type Packet struct {
	Q byte
	A int32
	B int32
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	log.Printf("2_meanstoanend at=handle-connection.start remote-addr=%q\n", conn.RemoteAddr())

	// Read through connection bytes
	for {
		// Add timeout to the connection to workaround binary.Read hanging forever on a closed connection
		conn.SetDeadline(time.Now().Add(time.Second))

		var p Packet
		err := binary.Read(conn, binary.BigEndian, &p)
		if err != nil && !errors.Is(err, io.EOF) {
			fmt.Println(err)
			break
		}

		switch p.Q {
		case 'I':
			s.handleInsert(p.A, p.B)
		case 'Q':
			mean := s.handleQuery(p.A, p.B)
			err := binary.Write(conn, binary.BigEndian, &mean)
			if err != nil && !errors.Is(err, io.EOF) {
				fmt.Println(err)
				break
			}
		default:
		}
	}

	log.Printf("2_meanstoanend at=handle-connection.finish remote-addr=%q\n", conn.RemoteAddr())
}

func (s *Server) handleInsert(timestamp, price int32) {
	log.Printf("2_meanstoanend at=handle-insert timestamp=%d price=%d\n", timestamp, price)

	s.Lock()
	defer s.Unlock()

	s.timestamps = sortedInsert(s.timestamps, timestamp)
	s.values[timestamp] = price
}

func (s *Server) handleQuery(mintime, maxtime int32) int32 {
	s.Lock()
	defer s.Unlock()

	count := 0
	sum := int32(0)
	for _, t := range s.timestamps {
		if mintime <= t && t <= maxtime {
			count += 1
			sum += s.values[t]
		}
	}
	log.Printf("2_meanstoanend at=handle-query mintime=%d maxtime=%d sum=%d count=%d\n", mintime, maxtime, sum, count)
	if count == 0 {
		return 0
	}
	return int32(float64(sum) / float64(count))
}

func sortedInsert(ss []int32, s int32) []int32 {
	i := 0
	for ; i < len(ss); i++ {
		if ss[i] >= s {
			break
		}
	}
	ss = append(ss, 0)
	copy(ss[i+1:], ss[i:])
	ss[i] = s
	return ss
}
