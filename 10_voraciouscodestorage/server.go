package voraciouscodestorage

import (
	"context"
	"errors"
	"fmt"
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
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(r)
		}
	}()
	defer conn.Close()

	log.Printf("10_voraciouscodestorage at=start addr=%q\n", conn.RemoteAddr().String())

	fmt.Fprintf(conn, "READY\n")

	scanner := NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		log.Printf("10_voraciouscodestorage at=input line=%q\n", line)

		// All commands must have at least one field, otherwise close connection
		if len(fields) == 0 {
			fmt.Fprintf(conn, "ERR illegal method:\n")
			return
		}

		switch strings.ToUpper(fields[0]) {
		case "HELP":
			fmt.Fprintf(conn, "OK usage: HELP|GET|PUT|LIST\n")
			fmt.Fprintf(conn, "READY\n")
		case "LIST":
			// if operand is not ascii, return error
			if !isASCII(fields[1]) || fields[1][0] != '/' {
				fmt.Fprintf(conn, "ERR illegal dir name\n")
				continue
			}
			s.storageMutex.RLock()
			files := listDir(s.storage, fields[1])
			s.storageMutex.RUnlock()

			sort.Strings(files)
			fmt.Fprintf(conn, "OK %d\n", len(files))
			for _, f := range files {
				fmt.Fprintf(conn, "%s\n", f)
			}
			fmt.Fprintf(conn, "READY\n")
		case "PUT":
			if len(fields) != 3 {
				fmt.Fprintf(conn, "ERR usage: PUT file length newline data\n")
				continue
			}
			if !isASCII(fields[1]) || fields[1][0] != '/' {
				fmt.Fprintf(conn, "ERR illegal file name\n")
				continue
			}
			length, err := strconv.Atoi(fields[2])
			if err != nil {
				fmt.Fprintf(conn, "ERR illegal file length\n")
				continue
			}
			if length < 0 {
				fmt.Fprintf(conn, "ERR illegal file length\n")
				continue
			}

			// Read file data
			data := make([]byte, length)
			fmt.Printf("Reading %d bytes\n", len(data))
			n, err := scanner.ReadFull(data)
			if err != nil || n != length {
				fmt.Printf("err=%s n=%d\n", err, n)
				fmt.Fprintf(conn, "ERR reading file data\n")
				continue
			}

			// Store file data
			s.storageMutex.Lock()
			s.storage[fields[1]] = append(s.storage[fields[1]], data)
			revision := len(s.storage[fields[1]])
			s.storageMutex.Unlock()

			fmt.Fprintf(conn, "OK r%d\n", revision)
			fmt.Fprintf(conn, "READY\n")

		case "GET":
			if len(fields) != 2 {
				fmt.Fprintf(conn, "ERR usage: GET file [revision]\n")
				continue
			}

			// if operand is not ascii, return error
			if !isASCII(fields[1]) || fields[1][0] != '/' {
				fmt.Fprintf(conn, "ERR illegal file name\n")
				continue
			}

			var revision int
			if len(fields) == 3 {
				r, err := strconv.Atoi(fields[2])
				if err != nil {
					fmt.Fprintf(conn, "ERR illegal revision\n")
					continue
				}
				if r < 0 {
					fmt.Fprintf(conn, "ERR illegal revision\n")
					continue
				}
				revision = r
			}

			// if file does not exist, return error

			s.storageMutex.RLock()
			revisions, ok := s.storage[fields[1]]
			s.storageMutex.RUnlock()

			if !ok {
				fmt.Fprintf(conn, "ERR file does not exist\n")
				continue
			}

			// if revision is specified, return that revision
			if len(fields) == 3 {
				if revision > len(revisions) {
					fmt.Fprintf(conn, "ERR revision does not exist\n")
					continue
				}
			} else {
				// otherwise return latest revision
				revision = len(revisions)
			}

			data := revisions[revision-1]
			fmt.Fprintf(conn, "OK %d\n", len(data))
			fmt.Fprintf(conn, "%s", data)
			fmt.Fprintf(conn, "READY\n")

		default:
			fmt.Fprintf(conn, "ERR illegal method: %s\n", fields[0])
			return
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("10_voraciouscodestorage at=handleConn err=%q\n", err)
	}

	log.Printf("10_voraciouscodestorage at=finish addr=%q\n", conn.RemoteAddr().String())
}

func isASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
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
