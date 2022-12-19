package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	budgetchat "github.com/fanatic/protohackers/5_mobinthemiddle"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "10005"
	}
	ctx := context.Background()

	s, err := budgetchat.NewServer(ctx, port)
	if err != nil {
		log.Fatalf("5_mobinthemiddle at=server err=%q\n", err)
	}

	done := make(chan struct{})
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		for sig := range c {
			log.Printf("5_mobinthemiddle at=server.exiting sig=%q\n", sig.String())
			s.Close()
			done <- struct{}{}
		}
	}()

	<-done
	log.Printf("5_mobinthemiddle at=server.finish\n")
}
