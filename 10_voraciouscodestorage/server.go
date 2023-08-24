package voraciouscodestorage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"

	proxyproto "github.com/pires/go-proxyproto"
)

type Server struct {
	Addr   string
	l      net.Listener
	cancel context.CancelFunc
	wg     sync.WaitGroup

	storage      map[string][][]byte
	storageMutex sync.RWMutex
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

	log.Printf("10_voraciouscodestorage at=server.listening addr=%q\n", l.Addr().String())
	s := &Server{Addr: l.Addr().String(), l: l, cancel: cancel, storage: map[string][][]byte{}}

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
				log.Printf("10_voraciouscodestorage at=accept err=%q\n", err)
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
	// defer func() {
	// 	if r := recover(); r != nil {
	// 		fmt.Println(r)
	// 	}
	// }()
	defer conn.Close()

	fmt.Fprintf(conn, "READY\n")

	scanner := NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		log.Printf("<-- %s\n", line)

		// All commands must have at least one field, otherwise close connection
		if len(fields) == 0 {
			replyf(conn, "ERR illegal method:")
			return
		}

		switch strings.ToUpper(fields[0]) {
		case "HELP":
			replyf(conn, "OK usage: HELP|GET|PUT|LIST")
			replyf(conn, "READY")
		case "LIST":
			if len(fields) != 2 {
				replyf(conn, "ERR usage: LIST dir")
				continue
			}

			// if operand is not ascii, return error
			if !isASCII(fields[1]) || fields[1][0] != '/' {
				replyf(conn, "ERR illegal dir name")
				continue
			}
			s.storageMutex.RLock()
			files := listDir(s.storage, fields[1])
			s.storageMutex.RUnlock()

			sort.Strings(files)
			replyf(conn, "OK %d", len(files))
			for _, f := range files {
				replyf(conn, "%s", f)
			}
			replyf(conn, "READY")
		case "PUT":
			if len(fields) != 3 {
				replyf(conn, "ERR usage: PUT file length newline data")
				continue
			}
			if !isASCII(fields[1]) || fields[1][0] != '/' {
				replyf(conn, "ERR illegal file name")
				continue
			}
			length, err := strconv.Atoi(fields[2])
			if err != nil {
				replyf(conn, "ERR illegal file length")
				continue
			}
			if length < 0 {
				replyf(conn, "ERR illegal file length")
				continue
			}

			// Read file data
			data := make([]byte, length)
			log.Printf("--- Reading %d bytes\n", len(data))
			n, err := scanner.ReadFull(data)
			if err != nil || n != length {
				fmt.Printf("Read %d bytes.  err=%s\n", n, err)
				replyf(conn, "ERR reading file data")
				continue
			}
			log.Printf("--- Read %d bytes\n", len(data))

			// check for content containing non-text character
			if !isText(string(data)) {
				fmt.Printf("Illegal file content: %q\n", string(data))
				replyf(conn, "ERR illegal file content")
				continue
			}

			// If latest revision matches, no need to store
			s.storageMutex.RLock()
			revisions, ok := s.storage[fields[1]]
			s.storageMutex.RUnlock()

			if ok && len(revisions) > 0 {
				log.Printf("--- Comparing %d (incoming) with %d (latest)\n", len(data), len(revisions[len(revisions)-1]))
				//log.Printf("--- %q\n", string(data))
				//log.Printf("--- %q\n", string(revisions[len(revisions)-1]))
				if string(revisions[len(revisions)-1]) == string(data) {
					replyf(conn, "OK r%d", len(revisions))
					replyf(conn, "READY")
					continue
				}
			}

			// Store file data
			s.storageMutex.Lock()
			s.storage[fields[1]] = append(s.storage[fields[1]], data)
			revision := len(s.storage[fields[1]])
			s.storageMutex.Unlock()

			replyf(conn, "OK r%d", revision)
			replyf(conn, "READY")

		case "GET":
			if len(fields) < 2 || len(fields) > 3 {
				replyf(conn, "ERR usage: GET file [revision]")
				continue
			}

			// if operand is not ascii, return error
			if !isASCII(fields[1]) || fields[1][0] != '/' {
				replyf(conn, "ERR illegal file name")
				continue
			}

			var revision int
			if len(fields) == 3 {
				r, err := strconv.Atoi(strings.TrimPrefix(fields[2], "r"))
				if err != nil {
					replyf(conn, "ERR illegal revision")
					continue
				}
				if r <= 0 {
					replyf(conn, "ERR illegal revision")
					continue
				}
				revision = r
			}

			// if file does not exist, return error

			s.storageMutex.RLock()
			revisions, ok := s.storage[fields[1]]
			s.storageMutex.RUnlock()

			if !ok {
				replyf(conn, "ERR file does not exist")
				continue
			}

			// if revision is specified, return that revision
			if len(fields) == 3 {
				if revision > len(revisions) {
					replyf(conn, "ERR revision does not exist")
					continue
				}
			} else {
				// otherwise return latest revision
				revision = len(revisions)
			}

			data := revisions[revision-1]
			replyf(conn, "OK %d", len(data))
			fmt.Fprintf(conn, "%s", data)
			replyf(conn, "READY")

		default:
			replyf(conn, "ERR illegal method: %s", fields[0])
			return
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("10_voraciouscodestorage at=handleConn err=%q\n", err)
	}
}

func replyf(w io.Writer, format string, args ...interface{}) {
	fmt.Fprintf(w, format+"\n", args...)
	log.Printf("--> %s\n", fmt.Sprintf(format, args...))
}

const validRunes = "/,-.0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwxyz"

func isASCII(s string) bool {
	for _, r := range s {
		if !strings.ContainsRune(validRunes, r) {
			return false
		}
	}
	return true
}

func isText(s string) bool {
	for _, r := range s {
		if (r < 0x20 || r > 0x7e) && r != '\n' && r != '\r' && r != '\t' {
			return false
		}
	}
	return true
}

func listDir(storage map[string][][]byte, dir string) []string {
	if dir[len(dir)-1] != '/' {
		dir += "/"
	}
	files := []string{}
	for f := range storage {
		if strings.HasPrefix(f, dir) {
			baseName := strings.TrimPrefix(f, dir)
			if strings.Contains(baseName, "/") {
				dirName := strings.Split(baseName, "/")[0] + "/"
				files = append(files, dirName+" DIR")
			} else {
				files = append(files, baseName+" r1")
			}
		}
	}
	return files
}
