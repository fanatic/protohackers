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

	t.Run("resend test", func(t *testing.T) {
		conn, err := net.Dial("udp", s.Addr)
		require.NoError(t, err)
		defer conn.Close()

		send(t, conn, "/connect/1/")
		expect(t, conn, "/ack/1/0/")

		send(t, conn, "/data/1/0/abcdefg/")
		expect(t, conn, "/ack/1/7/")

		send(t, conn, "/data/1/7/hijklmnop/")
		expect(t, conn, "/ack/1/16/")

		send(t, conn, "/data/1/5/01/")
		expect(t, conn, "/ack/1/16/")

		send(t, conn, "/data/1/13/23456789/")
		expect(t, conn, "/ack/1/21/")
	})

	t.Run("long lines with 25% loss", func(t *testing.T) {
		// NOTE:check starts
		// NOTE:checking whether long lines work (with 25% packet loss)
		// NOTE:successfully connected with session 692569413
		// FAIL:alarm timeout after 60 seconds

		conn, err := net.Dial("udp", s.Addr)
		require.NoError(t, err)
		defer conn.Close()

		send(t, conn, "/connect/692569413/")
		expect(t, conn, "/ack/692569413/0/")

		send(t, conn, "/data/692569413/2700/quartz now the jackdaws sphinx the quartz now of my for aid men party party quartz prisoners come giant PROTOHACKERS giant something to sphinx the royale aid men favicon royale for love the hypnotic prisoners giant party the to something of calculator love the of of royale giant prisoners to is love favicon bluebell love now nasa to my come aid integral of integral PROTOHACKERS love aid sphinx now intrusion jackdaws of about the of nasa to aid of the the bluebell royale love intrusion my casino hypnotic casino my of the all the party party integral to the to prisoners integral nasa hypnotic love time prisoners giant the hypnotic calculator my is integral to good for royale time about is for time party is the of of now giant nasa giant for love of giant the sphinx nasa my good my the hypnotic time my party of quartz all time love giant of favicon giant the casino is royale of to of bluebe/")
		expect(t, conn, "/ack/692569413/0/")

		send(t, conn, "/data/692569413/0/of royale of come quartz to now men love jackdaws to for good the hypnotic giant the hypnotic peach quartz all peach giant calculator the my party hypnotic sphinx the love calculator the of the intrusion for for to royale party integral the sphinx intrusion sphinx the giant intrusion the quartz nasa time aid prisoners of party giant nasa to jackdaws come giant hypnotic royale to quartz now intrusion party peach to for integral hypnotic bluebell for PROTOHACKERS bluebell favicon calculator quartz integral the giant all intrusion about of party something aid is to nasa jackdaws time about nasa of calculator time nasa to favicon sphinx is nasa hypnotic casino for about peach quartz nasa my my integral sphinx peach love to intrusion for integral jackdaws for now something PROTOHACKERS to integral good peach is aid giant to is sphinx men intrusion my the to peach love PROTOHACKERS hypnotic th/")
		// 2023-08-22T21:36:05Z app[1781327c5d7948] iad [info]2023/08/22 21:36:05 Pos: 0, old buffer: 0, new buffer: 900
		// 2023-08-22T21:36:05Z app[1781327c5d7948] iad [info]2023/08/22 21:36:05 B: "of royale of come quartz to now men love jackdaws to for good the hypnotic giant the hypnotic peach quartz all peach giant calculator the my party hypnotic sphinx the love calculator the of the intrusion for for to royale party integral the sphinx intrusion sphinx the giant intrusion the quartz nasa time aid prisoners of party giant nasa to jackdaws come giant hypnotic royale to quartz now intrusion party peach to for integral hypnotic bluebell for PROTOHACKERS bluebell favicon calculator quartz integral the giant all intrusion about of party something aid is to nasa jackdaws time about nasa of calculator time nasa to favicon sphinx is nasa hypnotic casino for about peach quartz nasa my my integral sphinx peach love to intrusion for integral jackdaws for now something PROTOHACKERS to integral good peach is aid giant to is sphinx men intrusion my the to peach love PROTOHACKERS hypnotic th"
		expect(t, conn, "/ack/692569413/900/")

		send(t, conn, "/data/692569413/900/e bluebell for party hypnotic is for favicon nasa casino to royale quartz now quartz about royale men is favicon aid intrusion royale for party to to now is something hypnotic jackdaws to time jackdaws my the calculator good my time all casino casino casino love jackdaws now about giant giant giant royale favicon all jackdaws peach casino casino sphinx intrusion prisoners all giant men for party jackdaws to bluebell to quartz the intrusion integral for the my casino something the my for giant of my giant calculator good to bluebell peach men about bluebell the for calculator calculator favicon the come favicon peach of of come my royale integral casino men peach my hypnotic now men to bluebell for now royale giant nasa jackdaws time intrusion come party nasa is intrusion is the my about favicon to nasa giant of the prisoners of to party the all giant sphinx peach prisoners jackdaws giant/")
		// 2023-08-22T21:36:05Z app[1781327c5d7948] iad [info]2023/08/22 21:36:05 Pos: 900, old buffer: 900, new buffer: 1800
		// 2023-08-22T21:36:05Z app[1781327c5d7948] iad [info]2023/08/22 21:36:05 B: "of royale of come quartz to now men love jackdaws to for good the hypnotic giant the hypnotic peach quartz all peach giant calculator the my party hypnotic sphinx the love calculator the of the intrusion for for to royale party integral the sphinx intrusion sphinx the giant intrusion the quartz nasa time aid prisoners of party giant nasa to jackdaws come giant hypnotic royale to quartz now intrusion party peach to for integral hypnotic bluebell for PROTOHACKERS bluebell favicon calculator quartz integral the giant all intrusion about of party something aid is to nasa jackdaws time about nasa of calculator time nasa to favicon sphinx is nasa hypnotic casino for about peach quartz nasa my my integral sphinx peach love to intrusion for integral jackdaws for now something PROTOHACKERS to integral good peach is aid giant to is sphinx men intrusion my the to peach love PROTOHACKERS hypnotic the bluebell for party hypnotic is for favicon nasa casino to royale quartz now quartz about royale men is favicon aid intrusion royale for party to to now is something hypnotic jackdaws to time jackdaws my the calculator good my time all casino casino casino love jackdaws now about giant giant giant royale favicon all jackdaws peach casino casino sphinx intrusion prisoners all giant men for party jackdaws to bluebell to quartz the intrusion integral for the my casino something the my for giant of my giant calculator good to bluebell peach men about bluebell the for calculator calculator favicon the come favicon peach of of come my royale integral casino men peach my hypnotic now men to bluebell for now royale giant nasa jackdaws time intrusion come party nasa is intrusion is the my about favicon to nasa giant of the prisoners of to party the all giant sphinx peach prisoners jackdaws giant"
		expect(t, conn, "/ack/692569413/1800/")

		send(t, conn, "/data/692569413/19800/tor the nasa giant of royale royale intrusion now time to PROTOHACKERS sphinx casino of about to all for peach aid aid of bluebell quartz casino now is giant of hypnotic peach giant giant the now prisoners love of peach come the of the party the party of men of of PROTOHACKERS to sphinx integral come to giant intrusion love\nthe something casino the men time favicon now favicon peach men giant about intrusion PROTOHACKERS party love about integral integral party quartz nasa now come to giant quartz of bluebell all sphinx good good casino of to something integral time bluebell giant men is the royale prisoners the men now prisoners aid favicon party of royale the PROTOHACKERS aid to royale the sphinx to about men party aid intrusion men giant time now jackdaws for bluebell peach for about aid to sphinx my of good casino integral nasa to the to casino men aid time for party prisoners is men/")
		expect(t, conn, "/ack/692569413/1800/")
		send(t, conn, "/data/692569413/0/of royale of come quartz to now men love jackdaws to for good the hypnotic giant the hypnotic peach quartz all peach giant calculator the my party hypnotic sphinx the love calculator the of the intrusion for for to royale party integral the sphinx intrusion sphinx the giant intrusion the quartz nasa time aid prisoners of party giant nasa to jackdaws come giant hypnotic royale to quartz now intrusion party peach to for integral hypnotic bluebell for PROTOHACKERS bluebell favicon calculator quartz integral the giant all intrusion about of party something aid is to nasa jackdaws time about nasa of calculator time nasa to favicon sphinx is nasa hypnotic casino for about peach quartz nasa my my integral sphinx peach love to intrusion for integral jackdaws for now something PROTOHACKERS to integral good peach is aid giant to is sphinx men intrusion my the to peach love PROTOHACKERS hypnotic th/")
		expect(t, conn, "/ack/692569413/1800/")
		send(t, conn, "/data/692569413/900/e bluebell for party hypnotic is for favicon nasa casino to royale quartz now quartz about royale men is favicon aid intrusion royale for party to to now is something hypnotic jackdaws to time jackdaws my the calculator good my time all casino casino casino love jackdaws now about giant giant giant royale favicon all jackdaws peach casino casino sphinx intrusion prisoners all giant men for party jackdaws to bluebell to quartz the intrusion integral for the my casino something the my for giant of my giant calculator good to bluebell peach men about bluebell the for calculator calculator favicon the come favicon peach of of come my royale integral casino men peach my hypnotic now men to bluebell for now royale giant nasa jackdaws time intrusion come party nasa is intrusion is the my about favicon to nasa giant of the prisoners of to party the all giant sphinx peach prisoners jackdaws giant/")
		expect(t, conn, "/ack/692569413/1800/")
		send(t, conn, "/data/692569413/1800/ to something sphinx good of sphinx the quartz to love giant good peach something hypnotic sphinx good is giant peach integral the to favicon to now all integral hypnotic of giant royale for of integral the now men giant jackdaws about to giant men nasa bluebell for nasa of sphinx love integral of the for nasa giant aid hypnotic love of the for sphinx party now now the good party prisoners the sphinx time about love hypnotic come peach peach giant my integral to giant time for the all for jackdaws the to the PROTOHACKERS nasa quartz to for the of now favicon to hypnotic time giant the party love PROTOHACKERS royale is favicon men is the calculator nasa calculator giant to love men royale nasa of bluebell come something time for integral integral party love bluebell favicon the peach calculator quartz hypnotic of calculator giant the all now good for the intrusion casino love hypnotic to /")
		expect(t, conn, "/ack/692569413/2700/")
		send(t, conn, "/data/692569413/3600/ll party intrusion time good sphinx for the all favicon the bluebell jackdaws something of jackdaws sphinx for now bluebell all party quartz is sphinx bluebell giant prisoners royale PROTOHACKERS party giant prisoners of the my intrusion love favicon love bluebell to nasa sphinx something party peach is royale of of peach my PROTOHACKERS good integral sphinx my PROTOHACKERS casino is hypnotic party jackdaws hypnotic calculator of come favicon the royale prisoners party prisoners time the of for of to giant nasa men the giant about the for party of peach men bluebell for prisoners for of of is the is of something now love all the my aid the to hypnotic of for time about royale to casino nasa jackdaws sphinx sphinx intrusion men time the hypnotic hypnotic to for giant prisoners aid giant time is time love men integral party my men giant the of party casino aid about aid something something/")
		expect(t, conn, "/ack/692569413/2700/")
	})

	t.Run("loss on receiver", func(t *testing.T) {
		t.Skip("blah")

		conn, err := net.Dial("udp", s.Addr)
		require.NoError(t, err)
		defer conn.Close()

		send(t, conn, "/connect/2/")
		expect(t, conn, "/ack/2/0/")

		send(t, conn, "/data/2/0/123\n/")

		go func() {
			for {
				send(t, conn, "/data/2/10/456\n/")
				time.Sleep(time.Millisecond * 100)
			}
		}()

		time.Sleep(30 * time.Second)
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
