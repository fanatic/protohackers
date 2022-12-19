package speeddaemon

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
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

	RoadLock    sync.Mutex
	Roads       map[uint16]Road    // road id
	SentTickets map[string][]int64 // plate to timestamps
}

type Road struct {
	Dispatcher     io.Writer
	DispatcherAddr string

	Limit   uint16
	Cameras map[uint16]Camera // location

	PendingTickets []Ticket
}

type Camera struct {
	Road     uint16
	Location uint16
	Camera   io.Writer

	SeenPlates map[string]uint32 // plate to timestamp
}

type Session struct {
	Dispatcher bool
	Camera     *Camera
	Heartbeat  bool

	c net.Conn
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

	log.Printf("6_speeddaemon at=server.listening addr=%q\n", l.Addr().String())
	s := &Server{Addr: l.Addr().String(), l: l, cancel: cancel, Roads: map[uint16]Road{}, SentTickets: map[string][]int64{}}

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
				log.Printf("6_speeddaemon at=accept err=%q\n", err)
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
	log.Printf("6_speeddaemon at=handle-connection.start remote-addr=%q\n", conn.RemoteAddr())

	sess := &Session{c: conn}

	for {
		err := s.handleMessage(sess)
		if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
			log.Printf("6_speeddaemon at=handle-connection.finish remote-addr=%q\n", conn.RemoteAddr())
			return
		} else if err != nil {
			log.Printf("6_speeddaemon at=handle-connection.err err=%s\n", err)
			return
		}
	}
}

func (s *Server) handleMessage(sess *Session) error {
	b := make([]byte, 1)
	n, err := sess.c.Read(b)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	} else if n != 1 && !errors.Is(err, io.EOF) {
		return fmt.Errorf("handleMessage: read %d characters, expected 1", n)
	} else if n != 1 && errors.Is(err, io.EOF) {
		return err
	}

	switch b[0] {
	case 0x20:
		return s.handleReadPlate(sess)
	case 0x40:
		return s.handleWantHeartbeat(sess)
	case 0x80:
		return s.handleIAmCamera(sess)
	case 0x81:
		return s.handleIAmDispatcher(sess)
	default:
		e := &Error{Msg: "bad message type"}
		e.Write(sess.c)
		return fmt.Errorf("bad message type")
	}
}

func (s *Server) handleReadPlate(sess *Session) error {
	plate, err := readString(sess.c)
	if err != nil {
		return err
	}

	timestamp, err := readUint32(sess.c)
	if err != nil {
		return err
	}

	log.Printf("6_speeddaemon <-- Plate{plate: %q, timestamp: %d}\n", plate, timestamp)

	if sess.Camera == nil {
		e := &Error{Msg: "not a camera"}
		e.Write(sess.c)
		return fmt.Errorf("not a camera")
	}
	c := sess.Camera
	s.RoadLock.Lock()
	c.SeenPlates[plate] = timestamp
	s.RoadLock.Unlock()

	s.checkSpeed(plate, c.Road)

	return nil
}

func (s *Server) handleWantHeartbeat(sess *Session) error {
	interval, err := readUint32(sess.c)
	if err != nil {
		return err
	}

	log.Printf("6_speeddaemon <-- Heartbeat{interval: %d}\n", interval)

	if sess.Heartbeat {
		e := &Error{Msg: "already heartbeating"}
		e.Write(sess.c)
		return fmt.Errorf("already heartbeating")
	}

	if interval <= 0 {
		return nil
	}

	sess.Heartbeat = true
	go func() {
		ticker := time.NewTicker(time.Duration(float64(interval)) * time.Second / 10)
		for range ticker.C {
			hb := &Heartbeat{}
			err := hb.Write(sess.c)
			if errors.Is(err, net.ErrClosed) {
				ticker.Stop()
				return
			} else if err != nil {
				log.Printf("6_speeddaemon at=heartbeat.err err=%s\n", err)
				ticker.Stop()
				return
			}
		}
	}()

	return nil
}

func (s *Server) handleIAmCamera(sess *Session) error {
	road, err := readUint16(sess.c)
	if err != nil {
		return err
	}

	mile, err := readUint16(sess.c)
	if err != nil {
		return err
	}

	limit, err := readUint16(sess.c)
	if err != nil {
		return err
	}

	log.Printf("6_speeddaemon <-- IAmCamera{road: %d, mile: %d, limit: %d}\n", road, mile, limit)

	if sess.Dispatcher {
		e := &Error{Msg: "already a dispatcher"}
		e.Write(sess.c)
		return fmt.Errorf("already a dispatcher")
	}

	if sess.Camera != nil {
		e := &Error{Msg: "already a camera"}
		e.Write(sess.c)
		return fmt.Errorf("already a camera")
	}

	s.RoadLock.Lock()
	r := s.Roads[road]
	r.Limit = limit
	if r.Cameras == nil {
		r.Cameras = map[uint16]Camera{}
	}
	c := r.Cameras[mile]
	c.Camera = sess.c
	c.Road = road
	c.Location = mile
	c.SeenPlates = map[string]uint32{}
	sess.Camera = &c
	r.Cameras[mile] = c
	s.Roads[road] = r
	s.RoadLock.Unlock()

	return nil
}

func (s *Server) handleIAmDispatcher(sess *Session) error {
	numroads, err := readUint8(sess.c)
	if err != nil {
		return err
	}

	roads := []uint16{}
	for i := 0; i < int(numroads); i++ {
		road, err := readUint16(sess.c)
		if err != nil {
			return err
		}
		roads = append(roads, road)
	}

	log.Printf("6_speeddaemon <-- IAmDispatcher{roads: %v}\n", roads)

	if sess.Camera != nil {
		e := &Error{Msg: "already a camera"}
		e.Write(sess.c)
		return fmt.Errorf("already a camera")
	}

	if sess.Dispatcher {
		e := &Error{Msg: "already a dispatcher"}
		e.Write(sess.c)
		return fmt.Errorf("already a dispatcher")
	}

	sess.Dispatcher = true

	s.RoadLock.Lock()
	for _, road := range roads {
		r := s.Roads[road]
		r.Dispatcher = sess.c
		r.DispatcherAddr = sess.c.RemoteAddr().String()

		// Send pending tickets for this road
		for _, t := range r.PendingTickets {
			if err := t.Write(sess.c); err != nil {
				return err
			}
		}
		r.PendingTickets = nil
		s.Roads[road] = r
	}
	s.RoadLock.Unlock()

	return nil
}

func (s *Server) checkSpeed(plate string, road uint16) {
	// s.RoadLock.Lock()
	// log.Printf("6_speeddaemon vvv CheckSpeed\n")
	// for road, r := range s.Roads {
	// 	log.Printf("6_speeddaemon Road{id: %d, dispatcher: %t, limit: %d, pending-tickets: %d}\n", road, r.Dispatcher != nil, r.Limit, len(r.PendingTickets))
	// 	for mile, c := range r.Cameras {
	// 		log.Printf("6_speeddaemon Camera{mile: %d, camera: %t, seen-plates: %v}\n", mile, c.Camera != nil, c.SeenPlates)
	// 	}
	// }
	// log.Printf("6_speeddaemon ^^^ CheckSpeed\n")
	// s.RoadLock.Unlock()

	s.RoadLock.Lock()
	r := s.Roads[road]
	observations := []Observation{}
	for mile, c := range r.Cameras {
		ts := c.SeenPlates[plate]
		observations = append(observations, Observation{Mile: mile, Timestamp: ts})
	}

	for i, o1 := range observations {
		for j, o2 := range observations {
			if i != j {
				if o1.Timestamp >= o2.Timestamp {
					continue
				}

				speed := speed(o1.Mile, o2.Mile, o1.Timestamp, o2.Timestamp)
				t := &Ticket{plate, road, o1.Mile, o1.Timestamp, o2.Mile, o2.Timestamp, speed}

				//log.Printf("6_speeddaemon !!! CompareObservations{plate: %q, road: %d, mile1: %d, timestamp1: %d, mile2: %d, timestamp2: %d, speed: %d, limit: %d}\n", t.Plate, t.Road, t.Mile1, t.Timestamp1, t.Mile2, t.Timestamp2, t.Speed, r.Limit)

				if uint16(math.Round(float64(t.Speed)/100)) > r.Limit {
					if s.checkAlreadyTicketed(plate, o1.Timestamp, o2.Timestamp) {
						s.RoadLock.Unlock()
						return
					}

					if r.Dispatcher != nil {
						log.Printf("6_speeddaemon !!! sending %s %s\n", plate, r.DispatcherAddr)
						if err := t.Write(r.Dispatcher); err != nil {
							log.Printf("6_speeddaemon at=ticket-write.err err=%s\n", err)
						}
					} else {
						log.Printf("6_speeddaemon !!! saving %s\n", plate)
						r.PendingTickets = append(r.PendingTickets, *t)
						s.Roads[road] = r
					}
				}
			}
		}
	}
	s.RoadLock.Unlock()
}

type Observation struct {
	Mile      uint16
	Timestamp uint32
}

func speed(m1, m2 uint16, t1, t2 uint32) uint16 {
	if t2 < t1 {
		return 0
	}

	distance := math.Abs(float64(m2) - float64(m1)) // miles
	time := float64(t2-t1) / 60 / 60                // hours

	speed := math.Round(distance / time * 100)
	//log.Printf("6_speeddaemon Speed{d: %f, t: %f, s: %f, r: %d\n", distance, time, speed, uint16(speed))

	// handle overflow
	if speed > 65535 {
		return 0
	}

	return uint16(speed)
}

func (s *Server) checkAlreadyTicketed(plate string, t1, t2 uint32) bool {
	day1 := int64(math.Floor(float64(t1) / 86400))
	day2 := int64(math.Floor(float64(t2) / 86400))

	seenDay1 := false
	seenDay2 := false
	for _, ts := range s.SentTickets[plate] {
		if day1 == day2 {
			if ts == day1 {
				log.Printf("6_speeddaemon !!! already ticketed %s %d %d %d\n", plate, ts, day1, day2)
				return true
			}
		} else {
			if ts == day1 || ts == day2 {
				log.Printf("6_speeddaemon !!! already ticketed %s %d %d (%d)\n", plate, ts, day1, day2)
				return true
			}
			// if seenDay2 && ts == day1 {
			// 	log.Printf("6_speeddaemon !!! already ticketed %s %d %d (%d)\n", plate, ts, day1, day2)
			// 	return true
			// } else if seenDay1 && ts == day2 {
			// 	log.Printf("6_speeddaemon !!! already ticketed %s %d (%d) %d\n", plate, ts, day1, day2)
			// 	return true
			// }
			if ts == day2 {
				seenDay2 = true
			}
			if ts == day1 {
				seenDay1 = true
			}
		}
	}

	// add both days
	if !seenDay1 {
		s.SentTickets[plate] = append(s.SentTickets[plate], day1)
	}
	if !seenDay2 && day1 != day2 {
		s.SentTickets[plate] = append(s.SentTickets[plate], day2)
	}
	log.Printf("6_speeddaemon !!! saving seen ticket %s %d %d\n", plate, day1, day2)
	return false
}
