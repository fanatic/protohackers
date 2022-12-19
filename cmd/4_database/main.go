package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	database "github.com/fanatic/protohackers/4_database"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = "fly-global-services:10004"
	}
	ctx := context.Background()

	s, err := database.NewServer(ctx, addr)
	if err != nil {
		log.Fatalf("4_database at=server err=%q\n", err)
	}

	done := make(chan struct{})
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		for sig := range c {
			log.Printf("4_database at=server.exiting sig=%q\n", sig.String())
			s.Close()
			done <- struct{}{}
		}
	}()

	<-done
	log.Printf("4_database at=server.finish\n")
}
