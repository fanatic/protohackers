package database

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
)

type Server struct {
	Addr   string
	l      net.PacketConn
	cancel context.CancelFunc
	wg     sync.WaitGroup

	dbLock sync.Mutex
	db     map[string]string
}

func NewServer(ctx context.Context, addr string) (*Server, error) {
	ctx, cancel := context.WithCancel(ctx)

	var lc net.ListenConfig
	l, err := lc.ListenPacket(ctx, "udp", addr)
	if err != nil {
		cancel()
		return nil, err
	}

	log.Printf("4_database at=server.listening addr=%q\n", l.LocalAddr().String())
	s := &Server{Addr: l.LocalAddr().String(), l: l, cancel: cancel, db: map[string]string{"version": "fanatic/protohackers"}}

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
	packet := make([]byte, 1000)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, addr, err := s.l.ReadFrom(packet)
			if errors.Is(err, net.ErrClosed) {
				return
			}
			if err != nil {
				log.Printf("4_database at=accept err=%q\n", err)
				continue
			}
			s.wg.Add(1)
			go func() {
				s.handlePacket(packet[:n], addr)
				s.wg.Done()
			}()
		}
	}
}

func (s *Server) handlePacket(packet []byte, addr net.Addr) {
	log.Printf("4_database at=handle-packet.start remote-addr=%q\n", addr)

	// handle insert
	if bytes.ContainsRune(packet, '=') {
		parts := bytes.SplitN(packet, []byte{'='}, 2)
		key, value := string(parts[0]), string(parts[1])

		// ignore attempts to modify version
		if key == "version" {
			log.Printf("4_database at=handle-packet.finish action=write-blocked key=%q remote-addr=%q\n", key, addr)
			return
		}

		s.dbLock.Lock()
		s.db[key] = value
		s.dbLock.Unlock()
		log.Printf("4_database at=handle-packet.finish action=write key=%q remote-addr=%q\n", key, addr)
		return
	}

	// handle retrieve
	key := string(packet)
	s.dbLock.Lock()
	value := s.db[key] // value can be empty string
	s.dbLock.Unlock()

	response := fmt.Sprintf("%s=%s", key, value)

	_, err := s.l.WriteTo([]byte(response), addr)
	if err != nil {
		log.Printf("4_database at=handle-packet.finish action=write-err key=%q remote-addr=%q err=%s\n", key, addr, err)
		return
	}
	log.Printf("4_database at=handle-packet.finish action=read key=%q remote-addr=%q\n", key, addr)
}
