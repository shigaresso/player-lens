package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	commandadapter "github.com/shigaresso/player-lens/cmd/collect-reviews/adapter"
	jsonoutput "github.com/shigaresso/player-lens/internal/jsonoutput/adapter"
	steamadapter "github.com/shigaresso/player-lens/internal/steam/adapter"
)

func main() { os.Exit(execute()) }

func execute() int {
	signal.Ignore(syscall.SIGPIPE)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	transport := http.DefaultTransport.(*http.Transport).Clone()
	defer transport.CloseIdleConnections()

	fetcher, err := steamadapter.NewClient(&http.Client{Transport: transport, Timeout: 10 * time.Second}, "")
	if err != nil {
		return commandadapter.Report(os.Stderr, err, 1)
	}

	saver, err := jsonoutput.NewSaver(os.Stdout)
	if err != nil {
		return commandadapter.Report(os.Stderr, err, 1)
	}

	return commandadapter.Run(ctx, os.Args[1:], fetcher, saver, os.Stderr)
}
