package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	budgetchat "github.com/fanatic/protohackers/6_speeddaemon"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "10006"
	}
	ctx := context.Background()

	s, err := budgetchat.NewServer(ctx, port)
	if err != nil {
		log.Fatalf("6_speeddaemon at=server err=%q\n", err)
	}

	done := make(chan struct{})
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		for sig := range c {
			log.Printf("6_speeddaemon at=server.exiting sig=%q\n", sig.String())
			s.Close()
			done <- struct{}{}
		}
	}()

	<-done
	log.Printf("6_speeddaemon at=server.finish\n")
}
