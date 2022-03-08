package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/jaeg/rocky-client/app"
)

func main() {
	app := &app.App{}
	err := app.Init()
	if err != nil {
		log.Fatal("Failed to init the app")
		os.Exit(1)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-c

		cancel()
	}()

	app.Run(ctx)
}
