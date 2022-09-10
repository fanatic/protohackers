package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	smoketest "github.com/fanatic/protohackers/0_smoketest"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}
	ctx := context.Background()

	s, err := smoketest.NewServer(ctx, port)
	if err != nil {
		log.Fatalf("0_smoketest at=server err=%q\n", err)
	}

	done := make(chan struct{})
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		for sig := range c {
			log.Printf("0_smoketest at=server.exiting sig=%q\n", sig.String())
			s.Close()
			done <- struct{}{}
		}
	}()

	<-done
	log.Printf("0_smoketest at=server.finish\n")
}
