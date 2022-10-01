package budgetchat

import (
	"bufio"
	"fmt"
	"net"
)

type Session struct {
	Name string
	Room *Room
	c    net.Conn
}

func NewSession(c net.Conn) (*Session, error) {
	s := &Session{
		c: c,
	}
	return s, nil
}

func (c *Session) ID() string {
	return c.c.RemoteAddr().String()
}

func (c *Session) Loop(r *Room) error {
	scanner := bufio.NewScanner(c.c)
	for scanner.Scan() {
		// Initial message is their name
		if c.Room == nil {
			c.Name = scanner.Text()
			if len(c.Name) == 0 {
				return fmt.Errorf("Username must contain at least 1 character")
			} else if !isAlphaNumeric(c.Name) {
				return fmt.Errorf("Username must consist entirely of alphanumeric characters")
			}
			r.Join(c)
		} else {
			msg := scanner.Text()
			c.Room.Broadcast(c, fmt.Sprintf("[%s] %s", c.Name, msg))
		}
	}
	return scanner.Err()
}

func (c *Session) SendMessage(s string) error {
	_, err := c.c.Write([]byte(s + "\n"))
	return err
}

func (c *Session) Close() {
	if c.Room != nil {
		c.Room.Leave(c)
	}
	c.c.Close()
}
