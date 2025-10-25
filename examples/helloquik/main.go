package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/ChizhovVadim/trader/pkg/connectors/quikservice"
)

func main() {
	var port int = 34132
	flag.IntVar(&port, "port", port, "")
	flag.Parse()

	var err = run(port)
	if err != nil {
		log.Println("app failed", "error", err)
	}
}

func run(port int) error {
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	var quikService = quikservice.New(log.Default(), port, 1)
	defer quikService.Close()
	var err = quikService.Init(ctx, nil)
	if err != nil {
		return err
	}

	data, err := quikService.IsConnected()
	if err != nil {
		return err
	}
	res, _ := quikservice.ParseInt(data.Data)
	fmt.Println(res == 1)

	data, err = quikService.MessageInfo("Где деньги, Лебовски?")
	if err != nil {
		return err
	}

	return nil
}
