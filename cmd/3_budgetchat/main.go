package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	budgetchat "github.com/fanatic/protohackers/3_budgetchat"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "10003"
	}
	ctx := context.Background()

	s, err := budgetchat.NewServer(ctx, port)
	if err != nil {
		log.Fatalf("3_budgetchat at=server err=%q\n", err)
	}

	done := make(chan struct{})
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		for sig := range c {
			log.Printf("3_budgetchat at=server.exiting sig=%q\n", sig.String())
			s.Close()
			done <- struct{}{}
		}
	}()

	<-done
	log.Printf("3_budgetchat at=server.finish\n")
}
