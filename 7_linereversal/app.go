package linereversal

import (
	"bufio"
	"io"
	"log"
	"strings"
)

func Handler(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(nil, 10*1024*1024)
	for scanner.Scan() {
		msg := scanner.Text()
		log.Printf("7_linereversal <== %s\n", msg)

		reversed := []byte(Reverse(msg))

		log.Printf("7_linereversal ==> %s\n", string(reversed))

		_, err := out.Write(append(reversed, '\n'))
		if err != nil {
			log.Printf("7_linereversal at=app.write.err err=%s\n", err)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("7_linereversal at=app.read.err err=%s\n", err)
	}
}

func Reverse(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := len(s) - 1; i >= 0; i-- {
		b.WriteByte(s[i])
	}
	return b.String()
}
