package tests

import (
	"context"
	"net"
	"testing"

	pestcontrol "github.com/fanatic/protohackers/11_pestcontrol"
	"github.com/stretchr/testify/require"
)

func TestLevel11PestControl(t *testing.T) {
	ctx := context.Background()
	s, err := pestcontrol.NewServer(ctx, "")
	require.NoError(t, err)
	defer s.Close()

	client, err := net.Dial("tcp", s.Addr)
	require.NoError(t, err)
	defer client.Close()

	// --> Hello
	require.NoError(t, pestcontrol.HandleHello(client))

	// <-- Hello
	require.NoError(t, pestcontrol.WriteHello(client))

	a, err := NewAuthorityServerMock()
	require.NoError(t, err)
	pestcontrol.AuthorityAddr = a.l.Addr().String()

	// <-- SiteVisit{site:12345, populations:[{species:"long-tailed rat",count:20}]}
	require.NoError(t, pestcontrol.WriteSiteVisit(client, 12345, []pestcontrol.Observation{
		{Species: "long-tailed rat", Count: 20},
	}))

}

// Mock authority server
type authorityServerMock struct {
	t *testing.T
	l net.Listener
}

func NewAuthorityServerMock() (*authorityServerMock, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return nil, err
	}

	s := &authorityServerMock{l: l}
	go s.acceptLoop()
	return s, nil
}

func (s *authorityServerMock) acceptLoop() {
	for {
		client, err := s.l.Accept()
		if err != nil {
			return
		}
		go s.handleClient(client)
	}
}

func (s *authorityServerMock) handleClient(client net.Conn) {
	defer client.Close()

	// <-- Hello
	require.NoError(s.t, pestcontrol.WriteHello(client))

	// --> Hello
	require.NoError(s.t, pestcontrol.HandleHello(client))

	// // <-- DialAuthority{site:12345}
	// require.NoError(s.t, pestcontrol.HandleDialAuthority(client, 12345))

	// // --> TargetPopulations{site:12345, targets:[{species:"long-tailed rat",min:0,max:10}]}
	// require.NoError(s.t, pestcontrol.WriteTargetPopulations(client))

	// // <-- CreatePolicy{policy:{species:"long-tailed rat",action:"cull"}}
	// require.NoError(s.t, pestcontrol.HandleCreatePolicy(client))

	// // --> PolicyResult{policy:123}
	// require.NoError(s.t, pestcontrol.WritePolicyResult(client, 123))
}
