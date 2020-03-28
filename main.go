package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"runtime"
	"time"

	"github.com/ChizhovVadim/trader/quik"

	"golang.org/x/sync/errgroup"
)

var (
	ErrNotImplemented = errors.New("not implemented")
	flagClient        = flag.String("client", "", "client key")
	buildCommit       string
)

func main() {
	flag.Parse()

	config, err := loadConfig()
	if err != nil {
		fmt.Println("load config failed", err)
		return
	}

	clientIndex, err := findClientIndex(config)
	if err != nil {
		fmt.Println("load client failed", err)
		return
	}
	var client = config.Clients[clientIndex]

	var logFolder = path.Join(config.LogPath, client.Key)
	var dateName = time.Now().Format("2006-01-02")

	// main log
	fLog, err := os.OpenFile(path.Join(logFolder, dateName+".log"), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println("open log file failed", err)
		return
	}
	defer fLog.Close()
	var logger = log.New(io.MultiWriter(os.Stderr, fLog), client.Key+" ", log.LstdFlags)

	logger.Println("Application started.")
	defer logger.Println("Application closed.")

	logger.Println("Environment",
		"BuildCommit", buildCommit,
		"RuntimeVersion", runtime.Version())

	// quik message log
	fQuikLog, err := os.OpenFile(path.Join(logFolder, dateName+"quik.log"), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println("open log file failed", err)
		return
	}
	defer fQuikLog.Close()
	var quikLogger = log.New(fQuikLog, "", log.LstdFlags)

	// quik callback message log
	fQuikCallback, err := os.OpenFile(path.Join(logFolder, dateName+"quikcallback.log"), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println("open log file failed", err)
		return
	}
	defer fQuikCallback.Close()
	var quikCallbackLogger = log.New(fQuikCallback, "", log.LstdFlags)

	var quikService = quik.New(logger, quikLogger, quikCallbackLogger, client.Port)

	var advisor = &Advisor{
		logger: logger,
		url:    config.AdvisorUrl,
		httpClient: &http.Client{
			Timeout: 100 * time.Second,
		},
	}

	var strategy = &Strategy{
		logger:         logger,
		quikService:    quikService,
		client:         client,
		advisorService: advisor,
	}

	g, ctx := errgroup.WithContext(context.Background())

	g.Go(func() error {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c:
			logger.Println("Interrupt...")
			return errors.New("Interrupt")
		}
	})

	g.Go(func() error {
		return quikService.DialAndServe(ctx)
	})

	g.Go(func() error {
		return strategy.Run(ctx)
	})

	err = g.Wait()
	if err != nil {
		logger.Println(err)
		return
	}
}

func findClientIndex(config StrategySettings) (int, error) {
	if len(config.Clients) == 0 {
		return 0, errors.New("no client in config")
	}

	if len(config.Clients) == 1 {
		return 0, nil
	}

	var clientKey string
	if *flagClient != "" {
		clientKey = *flagClient
	} else {
		fmt.Printf("Enter client: ")
		fmt.Scanln(&clientKey)
	}

	for i := range config.Clients {
		if config.Clients[i].Key == clientKey {
			return i, nil
		}
	}

	return 0, errors.New("no client found")
}
