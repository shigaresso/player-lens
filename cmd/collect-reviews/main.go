package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shigaresso/player-lens/internal/jsonoutput"
	"github.com/shigaresso/player-lens/internal/reviewcli"
	"github.com/shigaresso/player-lens/internal/steam"
)

func main() { os.Exit(execute()) }

func execute() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	transport := http.DefaultTransport.(*http.Transport).Clone()
	defer transport.CloseIdleConnections()
	fetcher, err := steam.NewClient(&http.Client{Transport: transport, Timeout: 10 * time.Second}, "")
	if err != nil {
		return report(os.Stderr, err, 1)
	}
	saver, err := jsonoutput.NewSaver(os.Stdout)
	if err != nil {
		return report(os.Stderr, err, 1)
	}
	return run(ctx, os.Args[1:], fetcher, saver, os.Stderr)
}

func run(ctx context.Context, args []string, fetcher steam.PageFetcher, saver steam.CollectionSaver, stderr io.Writer) int {
	err := reviewcli.Run(ctx, args, fetcher, saver)
	if err == nil {
		return 0
	}
	if errors.Is(err, flag.ErrHelp) {
		if _, err := fmt.Fprint(stderr, reviewcli.Usage); err != nil {
			return 1
		}
		return 0
	}
	code := 1
	if errors.Is(err, reviewcli.ErrUsage) {
		code = 2
	}
	return report(stderr, err, code)
}

func report(stderr io.Writer, err error, code int) int {
	if _, err := fmt.Fprintln(stderr, err); err != nil {
		return 1
	}
	return code
}
