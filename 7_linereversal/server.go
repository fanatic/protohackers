package linereversal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"regexp"
	"strconv"
	"sync"
)

type Server struct {
	Addr   string
	l      net.PacketConn
	cancel context.CancelFunc
	wg     sync.WaitGroup

	SessionLock sync.Mutex
	Sessions    map[int]*Session
}

func NewServer(ctx context.Context, addr string) (*Server, error) {
	ctx, cancel := context.WithCancel(ctx)

	var lc net.ListenConfig
	l, err := lc.ListenPacket(ctx, "udp", addr)
	if err != nil {
		cancel()
		return nil, err
	}

	log.Printf("7_linereversal at=server.listening addr=%q\n", l.LocalAddr().String())
	s := &Server{
		Addr:     l.LocalAddr().String(),
		l:        l,
		cancel:   cancel,
		Sessions: map[int]*Session{},
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
				log.Printf("7_linereversal at=accept err=%q\n", err)
				continue
			}

			// Copy of packet because it was getting corrupt from the gofunc
			msg := append(packet[:n][:0:0], packet[:n]...)

			s.wg.Add(1)
			go func() {
				s.handlePacket(msg, addr)
				s.wg.Done()
			}()
		}
	}
}

func (s *Server) handlePacket(packet []byte, addr net.Addr) {
	log.Printf("7_linereversal <-- %q %s\n", packet, addr)

	// Packet contents must begin with a forward slash, end with a forward slash,
	// have a valid message type, and have the correct number of fields for the message type.
	if len(packet) < 2 {
		log.Printf("7_linereversal at=handle-packet.err remote-addr=%q too-small\n", addr)
		return
	} else if packet[0] != '/' {
		log.Printf("7_linereversal at=handle-packet.err remote-addr=%q missing-begin\n", addr)
		return
	} else if packet[len(packet)-1] != '/' {
		log.Printf("7_linereversal at=handle-packet.err remote-addr=%q missing-end\n", addr)
		return
	}

	// Trim leading and trailing slash
	packet = packet[1:]
	packet = packet[:len(packet)-1]

	parts := bytes.SplitN(packet, []byte{'/'}, -1)
	if len(parts) < 1 {
		log.Printf("7_linereversal at=handle-packet.err remote-addr=%q parts=%d\n", addr, len(parts))
		return
	}

	msgType := string(parts[0])
	switch msgType {
	case "connect":
		if len(parts) != 2 {
			log.Printf("7_linereversal at=handle-packet.err remote-addr=%q parts=%d type=%s\n", addr, len(parts), msgType)
			return
		}
		session, _ := strconv.Atoi(string(parts[1]))
		s.handleConnect(session, addr)
	case "data":
		if len(parts) < 4 {
			log.Printf("7_linereversal at=handle-packet.err remote-addr=%q parts=%d type=%s\n", addr, len(parts), msgType)
			return
		}
		session, _ := strconv.Atoi(string(parts[1]))
		pos, _ := strconv.Atoi(string(parts[2]))

		parts := bytes.SplitN(packet, []byte{'/'}, 4) // re-split since DATA can contain escaped slashes
		s.handleData(session, pos, parts[3], addr)
	case "ack":
		if len(parts) != 3 {
			log.Printf("7_linereversal at=handle-packet.err remote-addr=%q parts=%d type=%s\n", addr, len(parts), msgType)
			return
		}
		session, _ := strconv.Atoi(string(parts[1]))
		length, _ := strconv.Atoi(string(parts[2]))
		s.handleAck(session, length, addr)
	case "close":
		if len(parts) != 2 {
			log.Printf("7_linereversal at=handle-packet.err remote-addr=%q parts=%d type=%s\n", addr, len(parts), msgType)
			return
		}
		session, _ := strconv.Atoi(string(parts[1]))
		s.handleClose(session, addr)
	}
}

func (s *Server) handleConnect(session int, remote net.Addr) {
	s.SessionLock.Lock()
	defer s.SessionLock.Unlock()

	_, exists := s.Sessions[session]
	if !exists {
		// If no session with this token is open: open one, and associate it
		// with the IP address and port number that the UDP packet originated from.
		s.Sessions[session] = NewSession(s, session, remote)
	}

	s.Reply(fmt.Sprintf("/ack/%d/0/", session), remote)
}

func (s *Server) handleData(session, pos int, data []byte, remote net.Addr) {
	s.SessionLock.Lock()
	defer s.SessionLock.Unlock()

	sess, exists := s.Sessions[session]
	if !exists {
		// If the session is not open: send /close/SESSION/ and stop.
		log.Printf("7_linereversal at=data.err session-missing\n")
		s.Reply(fmt.Sprintf("/close/%d/", session), remote)
		return
	}

	if len(matchUnescapedForwardSlashes.FindAllIndex(data, -1)) > 0 {
		log.Printf("7_linereversal at=handle-data.err remote-addr=%q data=%q found-unescaped-slashes\n", remote, data)
		return
	}

	// unescape
	data = bytes.ReplaceAll(data, []byte{'\\', '/'}, []byte{'/'})   // unescape forward slash \/ -> /
	data = bytes.ReplaceAll(data, []byte{'\\', '\\'}, []byte{'\\'}) // unescape back slash \\ -> \

	sess.Lock()
	defer sess.Unlock()

	lengthReceived := len(sess.buffer)

	if lengthReceived < pos {
		// Not received everything up to POS; send a duplicate of previous ack
		s.Reply(fmt.Sprintf("/ack/%d/%d/", session, lengthReceived), remote)
		return
	}

	// Insert length elements at position
	sess.buffer = append(sess.buffer[:pos], data...)

	log.Printf("Pos: %d, old buffer: %d, new buffer: %d\n", pos, lengthReceived, len(sess.buffer))
	log.Printf("B: %q\n", sess.buffer)

	s.Reply(fmt.Sprintf("/ack/%d/%d/", session, len(sess.buffer)), remote)

	// Pass up to application layer
	if len(sess.buffer) > lengthReceived {
		_, err := sess.AppIn.Write(sess.buffer[lengthReceived:])
		if err != nil {
			log.Printf("7_linereversal at=app-write err=%s\n", err)
			return
		}
	}
}

func (s *Server) handleAck(session, length int, remote net.Addr) {
	s.SessionLock.Lock()
	defer s.SessionLock.Unlock()

	sess, exists := s.Sessions[session]
	if !exists {
		// If the session is not open: send /close/SESSION/ and stop.
		log.Printf("7_linereversal at=data.err session-missing\n")
		s.Reply(fmt.Sprintf("/close/%d/", session), remote)
		return
	}

	sess.Lock()
	defer sess.Unlock()
	if length < sess.LargestAckLength {
		// do nothing and stop (assume it's a duplicate ack that got delayed).
		log.Printf("7_linereversal dropping duplicate ack %d < %d\n", length, sess.LargestAckLength)
		return
	} else if length > len(sess.OutBuffer) {
		// If the session is not open: send /close/SESSION/ and stop.
		log.Printf("7_linereversal at=data.err misbehaving-peer %d > %d\n", length, len(sess.OutBuffer))
		s.Reply(fmt.Sprintf("/close/%d/", session), remote)
		return
	} else if length < len(sess.OutBuffer) {
		// retransmit all payload data after the first LENGTH bytes.
		log.Printf("7_linereversal retransmitting %d < %d\n", length, len(sess.OutBuffer))

		sess.LargestAckLength = length
		sess.Unlock()
		sess.resendBuffer()
		sess.Lock()
	} else {
		if sess.LargestAckLength < length {
			sess.LargestAckLength = length
			log.Printf("7_linereversal updated ack to %d\n", sess.LargestAckLength)
		}
	}
}

func (s *Server) handleClose(session int, remote net.Addr) {
	s.SessionLock.Lock()
	defer s.SessionLock.Unlock()

	_, exists := s.Sessions[session]
	if !exists {
		// If the session is not open: send /close/SESSION/ and stop.
		log.Printf("7_linereversal at=data.err session-missing\n")
		s.Reply(fmt.Sprintf("/close/%d/", session), remote)
		return
	}

	delete(s.Sessions, session)
	s.Reply(fmt.Sprintf("/close/%d/", session), remote)
}

func (s *Server) Reply(response string, addr net.Addr) {
	if len(response) >= 1000 {
		log.Printf("7_linereversal at=reply.err remote-addr=%q too-long\n", addr.String())
		return
	}
	_, err := s.l.WriteTo([]byte(response), addr)
	if err != nil {
		log.Printf("7_linereversal at=reply.err remote-addr=%q err=%s\n", addr.String(), err)
		return
	}
	log.Printf("7_linereversal --> %q\n", response)
}

var matchUnescapedForwardSlashes = regexp.MustCompile(`([^\\]|^)\/`)
