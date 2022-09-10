package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	meanstoanend "github.com/fanatic/protohackers/2_meanstoanend"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "10002"
	}
	ctx := context.Background()

	s, err := meanstoanend.NewServer(ctx, port)
	if err != nil {
		log.Fatalf("2_meanstoanend at=server err=%q\n", err)
	}

	done := make(chan struct{})
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		for sig := range c {
			log.Printf("2_meanstoanend at=server.exiting sig=%q\n", sig.String())
			s.Close()
			done <- struct{}{}
		}
	}()

	<-done
	log.Printf("2_meanstoanend at=server.finish\n")
}
