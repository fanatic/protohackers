package linereversal

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Session struct {
	Session  int
	Remote   net.Addr
	LastSeen time.Time

	InLock sync.Mutex
	buffer []byte

	OutLock          sync.Mutex
	LargestAckLength int
	OutBuffer        []byte

	AppIn io.Writer

	s *Server
}

func NewSession(s *Server, session int, remote net.Addr) *Session {
	pr, pw := io.Pipe()

	sess := Session{
		Session:          session,
		Remote:           remote,
		LastSeen:         time.Now(),
		LargestAckLength: 0,
		AppIn:            pw,
		s:                s,
	}

	// "Boot" App
	go Handler(pr, &sess)
	go sess.Retrier()

	return &sess
}

func (sess *Session) Write(p []byte) (int, error) {
	sess.sendData(p)
	return len(p), nil
}

func (sess *Session) sendData(p []byte) {
	sess.OutLock.Lock()
	defer sess.OutLock.Unlock()

	// Split into 900 character chunks
	for low := 0; low < len(p)-1; low += 900 {
		l := 900
		if len(p)-low < 900 {
			l = len(p) - low
		}
		chunk := p[low : low+l]

		// escape
		data := bytes.ReplaceAll(chunk, []byte{'\\'}, []byte{'\\', '\\'}) // escape back slash \ -> \\
		data = bytes.ReplaceAll(data, []byte{'/'}, []byte{'\\', '/'})     // escape forward slash  / -> \/

		sess.s.Reply(fmt.Sprintf("/data/%d/%d/%s/", sess.Session, len(sess.OutBuffer), data), sess.Remote)
		sess.OutBuffer = append(sess.OutBuffer, chunk...)
	}
}

func (sess *Session) resendBuffer() {
	sess.OutLock.Lock()
	defer sess.OutLock.Unlock()

	if sess.LargestAckLength >= len(sess.OutBuffer) {
		return
	}

	pos := sess.LargestAckLength
	p := sess.OutBuffer[pos:]

	// Split into 900 character chunks
	for low := 0; low < len(p)-1; low += 900 {
		l := 900
		if len(p)-low < 900 {
			l = len(p) - low
		}
		chunk := p[low : low+l]

		// escape
		data := bytes.ReplaceAll(chunk, []byte{'\\'}, []byte{'\\', '\\'}) // escape back slash \ -> \\
		data = bytes.ReplaceAll(data, []byte{'/'}, []byte{'\\', '/'})     // escape forward slash  / -> \/

		sess.s.Reply(fmt.Sprintf("/data/%d/%d/%s/", sess.Session, pos, data), sess.Remote)
		pos += len(data)
	}
}

func (sess *Session) Retrier() {
	tick := time.NewTicker(3 * time.Second)
	for range tick.C {
		sinceLastSeen := time.Since(sess.LastSeen)

		if sinceLastSeen > 30*time.Second {
			tick.Stop()
			return
		}

		// retransmit all payload data after the largest ack length
		sess.resendBuffer()
	}
}
