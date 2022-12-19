package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	budgetchat "github.com/fanatic/protohackers/8_insecuresocketslayer"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "10008"
	}
	ctx := context.Background()

	s, err := budgetchat.NewServer(ctx, port)
	if err != nil {
		log.Fatalf("8_insecuresocketslayer at=server err=%q\n", err)
	}

	done := make(chan struct{})
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		for sig := range c {
			log.Printf("8_insecuresocketslayer at=server.exiting sig=%q\n", sig.String())
			s.Close()
			done <- struct{}{}
		}
	}()

	<-done
	log.Printf("8_insecuresocketslayer at=server.finish\n")
}
