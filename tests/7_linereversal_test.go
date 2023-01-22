package tests

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	linereversal "github.com/fanatic/protohackers/7_linereversal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLevel7LineReversal(t *testing.T) {
	ctx := context.Background()
	s, err := linereversal.NewServer(ctx, "")
	require.NoError(t, err)
	defer s.Close()

	t.Run("happy-path", func(t *testing.T) {
		conn, err := net.Dial("udp", s.Addr)
		require.NoError(t, err)
		defer conn.Close()

		send(t, conn, "/connect/12345/")
		expect(t, conn, "/ack/12345/0/")

		send(t, conn, "/data/12345/0/hello\n/")
		expect(t, conn, "/ack/12345/6/")
		expect(t, conn, "/data/12345/0/olleh\n/")
		send(t, conn, "/ack/12345/6/")

		// sometimes this ack reaches the server after the next data packet, so things get out of order
		// sleep between two sends..

		time.Sleep(time.Millisecond)

		send(t, conn, "/data/12345/6/Hello, world!\n/")
		expect(t, conn, "/ack/12345/20/")

		expect(t, conn, "/data/12345/6/!dlrow ,olleH\n/")
		send(t, conn, "/ack/12345/20/")

		send(t, conn, "/close/12345/")
		expect(t, conn, "/close/12345/")
	})

	t.Run("Simple slash", func(t *testing.T) {
		conn, err := net.Dial("udp", s.Addr)
		require.NoError(t, err)
		defer conn.Close()

		send(t, conn, "/connect/1234568/")
		expect(t, conn, "/ack/1234568/0/")

		send(t, conn, "/data/1234568/0/\\//")
		// note: 1, not 2, because the sequence "\/" only represents 1 byte of data
		expect(t, conn, "/ack/1234568/1/")

		send(t, conn, "/close/1234568/")
		expect(t, conn, "/close/1234568/")
	})

	t.Run("expect-retransmission", func(t *testing.T) {
		conn, err := net.Dial("udp", s.Addr)
		require.NoError(t, err)
		defer conn.Close()

		send(t, conn, "/connect/12345/")
		expect(t, conn, "/ack/12345/0/")

		send(t, conn, "/data/12345/0/hello\n/")
		expect(t, conn, "/ack/12345/6/")
		expect(t, conn, "/data/12345/0/olleh\n/")
		expect(t, conn, "/data/12345/0/olleh\n/")
		send(t, conn, "/ack/12345/6/")

		send(t, conn, "/data/12345/6/hello\n/")
		expect(t, conn, "/ack/12345/12/")
		expect(t, conn, "/data/12345/6/olleh\n/")
		expect(t, conn, "/data/12345/6/olleh\n/")
		send(t, conn, "/ack/12345/12/")

		send(t, conn, "/close/12345/")
		expect(t, conn, "/close/12345/")
	})

	t.Run("failing test case", func(t *testing.T) {
		conn, err := net.Dial("udp", s.Addr)
		require.NoError(t, err)
		defer conn.Close()

		send(t, conn, "/connect/1968668192/")
		expect(t, conn, "/ack/1968668192/0/")

		send(t, conn, "/data/1968668192/0/good something jackdaws prisoners the the\ncome something the PRO/")
		expect(t, conn, "/ack/1968668192/64/")

		// <== good something jackdaws prisoners the the
		// ==> eht eht srenosirp swadkcaj gnihtemos doog

		expect(t, conn, "/data/1968668192/0/eht eht srenosirp swadkcaj gnihtemos doog\n/")

		send(t, conn, "/data/1968668192/0/good something jackdaws prisoners the the\ncome something the PRO/")
		expect(t, conn, "/ack/1968668192/64/")

		send(t, conn, "/data/1968668192/64/TOHACKERS casino intrusion the aid come to something jackdaws royale favic\n/")
		expect(t, conn, "/ack/1968668192/139/")

		// <== come something the PROTOHACKERS casino intrusion the aid come to something jackdaws royale favic
		// ==> civaf elayor swadkcaj gnihtemos ot emoc dia eht noisurtni onisac SREKCAHOTORP eht gnihtemos emoc

		// send(t, conn, "/close/1968668192/")
		// expect(t, conn, "/close/1968668192/")

	})

	t.Run("escaping", func(t *testing.T) {
		conn, err := net.Dial("udp", s.Addr)
		require.NoError(t, err)
		defer conn.Close()

		send(t, conn, "/connect/573772849/")
		expect(t, conn, "/ack/573772849/0/")

		send(t, conn, "/data/573772849/0/foo\\/bar\\/baz\nfoo\\\\bar\\\\baz\n/")
		expect(t, conn, "/ack/573772849/24/")

		// <== foo/bar/baz
		// ==> zab/rab/oof

		expect(t, conn, "/data/573772849/0/zab\\/rab\\/oof\n/")
		send(t, conn, "/ack/573772849/12/")
		expect(t, conn, "/data/573772849/12/zab\\\\rab\\\\oof\n/")
		send(t, conn, "/ack/573772849/24/")

		// <-- "/connect/573772849/" 206.189.113.124:48879
		// --> "/ack/573772849/0/"
		//
		// <-- "/data/573772849/0/foo\\/bar\\/baz\nfoo\\\\bar\\\\baz\n/" 206.189.113.124:48879
		// --> "/ack/573772849/26/"
		//
		//
		// --> "/data/573772849/0/zab\\/rab\\/oof\n/"
		//
		// <== foo\\bar\\baz
		// ==> zab\\rab\\oof
		//
		// --> "/data/573772849/12/zab\\\\rab\\\\oof\n/"
		//
		// <-- "/data/573772849/0/foo\\/bar\\/baz\nfoo\\\\bar\\\\baz\n/" 206.189.113.124:48879
		// --> "/ack/573772849/26/"
		//
		//
		// <-- "/close/573772849/" 206.189.113.124:48879
		// --> "/close/573772849/"
	})

	t.Run("block extra slashes", func(t *testing.T) {
		conn, err := net.Dial("udp", s.Addr)
		require.NoError(t, err)
		defer conn.Close()

		send(t, conn, "/connect/321/")
		expect(t, conn, "/ack/321/0/")

		send(t, conn, "/data/321/0/foo\\/bar/")
		expect(t, conn, "/ack/321/7/")

		send(t, conn, "/data/321/0/foo/bar/")
		// don't expect(t, conn, "/ack/573772849/7/")

	})

	t.Run("long lines", func(t *testing.T) {
		conn, err := net.Dial("udp", s.Addr)
		require.NoError(t, err)
		defer conn.Close()

		send(t, conn, "/connect/2451706/")
		expect(t, conn, "/ack/2451706/0/")

		send(t, conn, "/data/2451706/26100/usion love party hypnotic favicon integral sphinx PROTOHACKERS for the aid giant for of royale for is royale casino about to all integral casino party of time something good is the party to intrusion quartz calculator time for the to integral giant nasa giant prisoners giant all sphinx hypnotic intrusion come prisoners giant prisoners good for prisoners prisoners the the jackdaws to jackdaws party sphinx my to intrusion love giant sphinx to sphinx jackdaws sphinx of men peach to the integral royale integral the the giant casino bluebell about is jackdaws about bluebell of giant casino royale is now sphinx my something to something now favicon calculator bluebell about for hypnotic something the to aid is giant all something to to men casino the to peach to quartz calculator of to the time integral sphinx come casino casino quartz of of love quartz now of jackdaws time for PROTOHACKERS ab/")
		expect(t, conn, "/ack/2451706/0/")
	})

	t.Run("long lines", func(t *testing.T) {
		t.Skip("runs forever")
		conn, err := net.Dial("udp", s.Addr)
		require.NoError(t, err)
		defer conn.Close()

		send(t, conn, "/connect/2451706/")
		expect(t, conn, "/ack/2451706/0/")

		send(t, conn, "/data/2451706/0/usion love party hypnotic favicon integral sphinx PROTOHACKERS for the aid giant for of royale for is royale casino about to all integral casino party of time something good is the party to intrusion quartz calculator time for the to integral giant nasa giant prisoners giant all sphinx hypnotic intrusion come prisoners giant prisoners good for prisoners prisoners the the jackdaws to jackdaws party sphinx my to intrusion love giant sphinx to sphinx jackdaws sphinx of men peach to the integral royale integral the the giant casino bluebell about is jackdaws about bluebell of giant casino royale is now sphinx my something to something now favicon calculator bluebell about for hypnotic something the to aid is giant all something to to men casino the to peach to quartz calculator of to the time integral sphinx come casino casino quartz of of love quartz now of jackdaws time for PROTOHACKERS ab/")
		expect(t, conn, "/ack/2451706/900/")
		send(t, conn, "/data/2451706/900/usion love party hypnotic favicon integral sphinx PROTOHACKERS for the aid giant for of royale for is royale casino about to all integral casino party of time something good is the party to intrusion quartz calculator time for the to integral giant nasa giant prisoners giant all sphinx hypnotic intrusion come prisoners giant prisoners good for prisoners prisoners the the jackdaws to jackdaws party sphinx my to intrusion love giant sphinx to sphinx jackdaws sphinx of men peach to the integral royale integral the the giant casino bluebell about is jackdaws about bluebell of giant casino royale is now sphinx my something to something now favicon calculator bluebell about for hypnotic something the to aid is giant all something to to men casino the to peach to quartz calculator of to the time integral sphinx come casino casino quartz of of love quartz now of jackdaws time for PROTOHACKERS ab/")
		expect(t, conn, "/ack/2451706/1800/")
		send(t, conn, "/data/2451706/1800/usion love party hypnotic favicon integral sphinx PROTOHACKERS for the aid giant for of royale for is royale casino about to all integral casino party of time something good is the party to intrusion quartz calculator time for the to integral giant nasa giant prisoners giant all sphinx hypnotic intrusion come prisoners giant prisoners good for prisoners prisoners the the jackdaws to jackdaws party sphinx my to intrusion love giant sphinx to sphinx jackdaws sphinx of men peach to the integral royale integral the the giant casino bluebell about is jackdaws about bluebell of giant casino royale is now sphinx my something to something now favicon calculator bluebell about for hypnotic something the to aid is giant all something to to men casino the to peach to quartz calculator of to the time integral sphinx come casino casino quartz of of love quartz now of jackdaws time for PROTOHACKERS ab/")
		expect(t, conn, "/ack/2451706/2700/")
		send(t, conn, "/data/2451706/2700/usion love party hypnotic favicon integral sphinx PROTOHACKERS for the aid giant for of royale for is royale casino about to all integral casino party of time something good is the party to intrusion quartz calculator time for the to integral giant nasa giant prisoners giant all sphinx hypnotic intrusion come prisoners giant prisoners good for prisoners prisoners the the jackdaws to jackdaws party sphinx my to intrusion love giant sphinx to sphinx jackdaws sphinx of men peach to the integral royale integral the the giant casino bluebell about is jackdaws about bluebell of giant casino royale is now sphinx my something to something now favicon calculator bluebell about for hypnotic something the to aid is giant all something to to men casino the to peach to quartz calculator of to the time integral sphinx come casino casino quartz of of love quartz now of jackdaws time for PROTOHACKERS ab/")
		expect(t, conn, "/ack/2451706/3600/")
		send(t, conn, "/data/2451706/3600/usion love party hypnotic favicon integral sphinx PROTOHACKERS for the aid giant for of royale for is royale casino about to all integral casino party of time something good is the party to intrusion quartz calculator time for the to integral giant nasa giant prisoners giant all sphinx hypnotic intrusion come prisoners giant prisoners good for prisoners prisoners the the jackdaws to jackdaws party sphinx my to intrusion love giant sphinx to sphinx jackdaws sphinx of men peach to the integral royale integral the the giant casino bluebell about is jackdaws about bluebell of giant casino royale is now sphinx my something to something now favicon calculator bluebell about for hypnotic something the to aid is giant all something to to men casino the to peach to quartz calculator of to the time integral sphinx come casino casino quartz of of love quartz now of jackdaws time for PROTOHACKERS ab/")
		expect(t, conn, "/ack/2451706/4500/")
		send(t, conn, "/data/2451706/4500/usion love party hypnotic favicon integral sphinx PROTOHACKERS for the aid giant for of royale for is royale casino about to all integral casino party of time something good is the party to intrusion quartz calculator time for the to integral giant nasa giant prisoners giant all sphinx hypnotic intrusion come prisoners giant prisoners good for prisoners prisoners the the jackdaws to jackdaws party sphinx my to intrusion love giant sphinx to sphinx jackdaws sphinx of men peach to the integral royale integral the the giant casino bluebell about is jackdaws about bluebell of giant casino royale is now sphinx my something to something now favicon calculator bluebell about for hypnotic something the to aid is giant all something to to men casino the to peach to quartz calculator of to the time integral sphinx come casino casino quartz of of love quartz now of jackdaws time for PROTOHACKERS ab/")
		expect(t, conn, "/ack/2451706/5400/")
		send(t, conn, "/data/2451706/5400/usion love party hypnotic favicon integral sphinx PROTOHACKERS for the aid giant for of royale for is royale casino about to all integral casino party of time something good is the party to intrusion quartz calculator time for the to integral giant nasa giant prisoners giant all sphinx hypnotic intrusion come prisoners giant prisoners good for prisoners prisoners the the jackdaws to jackdaws party sphinx my to intrusion love giant sphinx to sphinx jackdaws sphinx of men peach to the integral royale integral the the giant casino bluebell about is jackdaws about bluebell of giant casino royale is now sphinx my something to something now favicon calculator bluebell about for hypnotic something the to aid is giant all something to to men casino the to peach to quartz calculator of to the time integral sphinx come casino casino quartz of of love quartz now of jackdaws time for PROTOHACKERS ab/")
		expect(t, conn, "/ack/2451706/6300/")
		send(t, conn, "/data/2451706/6300/usion love party hypnotic favicon integral sphinx PROTOHACKER\n for the aid giant for of royale for is royale casino about to all integral casino party of time something good is the party to intrusion quartz calculator time for the to integral giant nasa giant prisoners giant all sphinx hypnotic intrusion come prisoners giant prisoners good for prisoners prisoners the the jackdaws to jackdaws party sphinx my to intrusion love giant sphinx to sphinx jackdaws sphinx of men peach to the integral royale integral the the giant casino bluebell about is jackdaws about bluebell of giant casino royale is now sphinx my something to something now favicon calculator bluebell about for hypnotic something the to aid is giant all something to to men casino the to peach to quartz calculator of to the time integral sphinx come casino casino quartz of of love quartz now of jackdaws time for PROTOHACKERS ab/")
		expect(t, conn, "/ack/2451706/7200/")
		send(t, conn, "/data/2451706/7200/usion love party hypnotic favicon integral sphinx PROTOHACKER\n for the aid giant for of royale for is royale casino about to all integral casino party of time something good is the party to intrusion quartz calculator time for the to integral giant nasa giant prisoners giant all sphinx hypnotic intrusion come prisoners giant prisoners good for prisoners prisoners the the jackdaws to jackdaws party sphinx my to intrusion love giant sphinx to sphinx jackdaws sphinx of men peach to the integral royale integral the the giant casino bluebell about is jackdaws about bluebell of giant casino royale is now sphinx my something to something now favicon calculator bluebell about for hypnotic something the to aid is giant all something to to men casino the to peach to quartz calculator of to the time integral sphinx come casino casino quartz of of love quartz now of jackdaws time for PROTOHACKERS ab/")
		expect(t, conn, "/ack/2451706/8100/")
		send(t, conn, "/data/2451706/8100/usion love party hypnotic favicon integral sphinx PROTOHACKER\n for the aid giant for of royale for is royale casino about to all integral casino party of time something good is the party to intrusion quartz calculator time for the to integral giant nasa giant prisoners giant all sphinx hypnotic intrusion come prisoners giant prisoners good for prisoners prisoners the the jackdaws to jackdaws party sphinx my to intrusion love giant sphinx to sphinx jackdaws sphinx of men peach to the integral royale integral the the giant casino bluebell about is jackdaws about bluebell of giant casino royale is now sphinx my something to something now favicon calculator bluebell about for hypnotic something the to aid is giant all something to to men casino the to peach to quartz calculator of to the time integral sphinx come casino casino quartz of of love quartz now of jackdaws time for PROTOHACKERS ab/")
		expect(t, conn, "/ack/2451706/9000/")
		send(t, conn, "/data/2451706/9000/usion love party hypnotic favicon integral sphinx PROTOHACKER\n for the aid giant for of royale for is royale casino about to all integral casino party of time something good is the party to intrusion quartz calculator time for the to integral giant nasa giant prisoners giant all sphinx hypnotic intrusion come prisoners giant prisoners good for prisoners prisoners the the jackdaws to jackdaws party sphinx my to intrusion love giant sphinx to sphinx jackdaws sphinx of men peach to the integral royale integral the the giant casino bluebell about is jackdaws about bluebell of giant casino royale is now sphinx my something to something now favicon calculator bluebell about for hypnotic something the to aid is giant all something to to men casino the to peach to quartz calculator of to the time integral sphinx come casino casino quartz of of love quartz now of jackdaws time for PROTOHACKERS ab/")
		expect(t, conn, "/ack/2451706/9900/")
		send(t, conn, "/data/2451706/9900/usion love party hypnotic favicon integral sphinx PROTOHACKER\n for the aid giant for of royale for is royale casino about to all integral casino party of time something good is the party to intrusion quartz calculator time for the to integral giant nasa giant prisoners giant all sphinx hypnotic intrusion come prisoners giant prisoners good for prisoners prisoners the the jackdaws to jackdaws party sphinx my to intrusion love giant sphinx to sphinx jackdaws sphinx of men peach to the integral royale integral the the giant casino bluebell about is jackdaws about bluebell of giant casino royale is now sphinx my something to something now favicon calculator bluebell about for hypnotic something the to aid is giant all something to to men casino the to peach to quartz calculator of to the time integral sphinx come casino casino quartz of of love quartz now of jackdaws time for PROTOHACKERS ab/")
		expect(t, conn, "/ack/2451706/10800/")

		expect(t, conn, "/data/2451706/0/REKCAHOTORP xnihps largetni nocivaf citonpyh ytrap evol noisuba SREKCAHOTORP rof emit swadkcaj fo won ztrauq evol fo fo ztrauq onisac onisac emoc xnihps largetni emit eht ot fo rotaluclac ztrauq ot hcaep ot eht onisac nem ot ot gnihtemos lla tnaig si dia ot eht gnihtemos citonpyh rof tuoba llebeulb rotaluclac nocivaf won gnihtemos ot gnihtemos ym xnihps won si elayor onisac tnaig fo llebeulb tuoba swadkcaj si tuoba llebeulb onisac tnaig eht eht largetni elayor largetni eht ot hcaep nem fo xnihps swadkcaj xnihps ot xnihps tnaig evol noisurtni ot ym xnihps ytrap swadkcaj ot swadkcaj eht eht srenosirp srenosirp rof doog srenosirp tnaig srenosirp emoc noisurtni citonpyh xnihps lla tnaig srenosirp tnaig asan tnaig largetni ot eht rof emit rotaluclac ztrauq noisurtni ot ytrap eht si doog gnihtemos emit fo ytrap onisac largetni lla ot tuoba onisac elayor si rof elayor fo rof tnaig dia eht rof S/")
		send(t, conn, "/ack/2451706/900/")

	})

}

func send(t *testing.T, conn net.Conn, msg string) {
	_, err := conn.Write([]byte(msg))
	require.NoError(t, err)
}

func expect(t *testing.T, conn net.Conn, msg string) {
	received := make([]byte, len(msg))
	n, err := io.ReadFull(conn, received)
	require.NoError(t, err)
	assert.Equal(t, msg, string(received[:n])) // will be truncated to expected message length
}
