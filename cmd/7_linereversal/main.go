package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	linereversal "github.com/fanatic/protohackers/7_linereversal"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = "fly-global-services:10007"
	}
	ctx := context.Background()

	s, err := linereversal.NewServer(ctx, addr)
	if err != nil {
		log.Fatalf("7_linereversal at=server err=%q\n", err)
	}

	done := make(chan struct{})
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		for sig := range c {
			log.Printf("7_linereversal at=server.exiting sig=%q\n", sig.String())
			s.Close()
			done <- struct{}{}
		}
	}()

	<-done
	log.Printf("7_linereversal at=server.finish\n")
}
