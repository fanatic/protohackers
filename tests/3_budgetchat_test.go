package tests

import (
	"bufio"
	"context"
	"net"
	"testing"

	budgetchat "github.com/fanatic/protohackers/3_budgetchat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLevel3BudgetChat(t *testing.T) {
	ctx := context.Background()
	s, err := budgetchat.NewServer(ctx, "")
	require.NoError(t, err)
	defer s.Close()

	t.Run("happy-path", func(t *testing.T) {
		bob, err := New(s.Addr)
		require.NoError(t, err)
		defer bob.Close()
		assert.Equal(t, "Welcome to budgetchat! What shall I call you?", bob.ReadMessage())
		require.NoError(t, bob.SendMessage("bob"))
		assert.Equal(t, "* The room is empty", bob.ReadMessage())

		charlie, err := New(s.Addr)
		require.NoError(t, err)
		defer charlie.Close()
		assert.Equal(t, "Welcome to budgetchat! What shall I call you?", charlie.ReadMessage())
		require.NoError(t, charlie.SendMessage("charlie"))
		assert.Equal(t, "* The room contains: bob", charlie.ReadMessage())

		dave, err := New(s.Addr)
		require.NoError(t, err)
		defer dave.Close()
		assert.Equal(t, "Welcome to budgetchat! What shall I call you?", dave.ReadMessage())
		require.NoError(t, dave.SendMessage("dave"))
		assert.Equal(t, "* The room contains: bob, charlie", dave.ReadMessage())

		alice, err := New(s.Addr)
		require.NoError(t, err)
		defer alice.Close()

		// --> Welcome to budgetchat! What shall I call you?
		assert.Equal(t, "Welcome to budgetchat! What shall I call you?", alice.ReadMessage())

		// <-- alice
		require.NoError(t, alice.SendMessage("alice"))

		// --> * The room contains: bob, charlie, dave
		assert.Equal(t, "* The room contains: bob, charlie, dave", alice.ReadMessage())

		// <-- Hello everyone
		require.NoError(t, alice.SendMessage("Hello everyone"))

		// --> [bob] hi alice
		require.NoError(t, bob.SendMessage("hi alice"))
		assert.Equal(t, "[bob] hi alice", alice.ReadMessage())

		// --> [charlie] hello alice
		require.NoError(t, charlie.SendMessage("hello alice"))
		assert.Equal(t, "[charlie] hello alice", alice.ReadMessage())

		// --> * dave has left the room
		dave.Close()
		assert.Equal(t, "* dave has left the room", alice.ReadMessage())

	})
}

type ChatClient struct {
	c    net.Conn
	msgs chan string
}

func New(addr string) (*ChatClient, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	c := &ChatClient{
		c:    conn,
		msgs: make(chan string),
	}
	go c.readerLoop()
	return c, err
}

func (c *ChatClient) readerLoop() error {
	scanner := bufio.NewScanner(c.c)
	for scanner.Scan() {
		c.msgs <- scanner.Text()
	}
	return scanner.Err()
}

func (c *ChatClient) ReadMessage() string {
	return <-c.msgs
}

func (c *ChatClient) SendMessage(s string) error {
	_, err := c.c.Write([]byte(s + "\n"))
	return err
}

func (c *ChatClient) Close() {
	c.c.Close()
}
