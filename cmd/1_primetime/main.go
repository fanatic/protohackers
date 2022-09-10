package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	primetime "github.com/fanatic/protohackers/1_primetime"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "10001"
	}
	ctx := context.Background()

	s, err := primetime.NewServer(ctx, port)
	if err != nil {
		log.Fatalf("1_primetime at=server err=%q\n", err)
	}

	done := make(chan struct{})
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		for sig := range c {
			log.Printf("1_primetime at=server.exiting sig=%q\n", sig.String())
			s.Close()
			done <- struct{}{}
		}
	}()

	<-done
	log.Printf("1_primetime at=server.finish\n")
}
