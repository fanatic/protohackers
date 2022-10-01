package budgetchat

import (
	"fmt"
	"log"
	"strings"
	"sync"
)

type Room struct {
	sync.Mutex
	sessions []Session
}

func NewRoom() *Room {
	return &Room{}
}

func (r *Room) Join(s *Session) {
	if len(r.sessions) > 0 {
		s.SendMessage(fmt.Sprintf("* The room contains: %s", names(r.sessions)))
	} else {
		s.SendMessage("* The room is empty")
	}

	r.Lock()
	r.sessions = append(r.sessions, *s)
	s.Room = r
	r.Unlock()

	r.Broadcast(s, fmt.Sprintf("* %s has entered the room", s.Name))
	log.Printf("3_budgetchat at=room.join name=%s\n", s.Name)
}

func names(sessions []Session) string {
	r := []string{}
	for _, s := range sessions {
		r = append(r, s.Name)
	}
	return strings.Join(r, ", ")
}

func (r *Room) Leave(s *Session) {
	r.Lock()
	r.sessions = removeSession(r.sessions, s)
	r.Unlock()

	r.Broadcast(s, fmt.Sprintf("* %s has left the room", s.Name))
	log.Printf("3_budgetchat at=room.leave name=%s\n", s.Name)
}

func removeSession(sessions []Session, sess *Session) []Session {
	filtered := []Session{}

	for _, s := range sessions {
		if s.ID() != sess.ID() {
			filtered = append(filtered, s)
		}
	}

	return filtered
}

func (r *Room) Broadcast(source *Session, msg string) {
	r.Lock()
	sessions := r.sessions
	r.Unlock()

	log.Printf("3_budgetchat at=room.msg name=%s msg=%d recp=%d\n", source.Name, len(msg), len(sessions))

	for _, s := range sessions {
		if source != nil && source.ID() == s.ID() {
			continue
		}
		log.Printf("3_budgetchat at=room.broadcast to=%s msg=%d\n", s.Name, len(msg))
		if err := s.SendMessage(msg); err != nil {
			log.Printf("3_budgetchat at=broadcast err=%q\n", err.Error())
		}
		log.Printf("3_budgetchat at=room.broadcast.done to=%s msg=%d\n", s.Name, len(msg))
	}

	log.Printf("3_budgetchat at=room.msg.done name=%s msg=%d recp=%d\n", source.Name, len(msg), len(sessions))
}
