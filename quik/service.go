package quik

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"time"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

type QuikService struct {
	logger         *log.Logger
	messageLogger  *log.Logger
	callbackLogger *log.Logger
	port           int
	queries        chan query
	transId        int64
	eventManager   EventManager
}

type query struct {
	command  string
	request  interface{}
	response interface{}
	done     chan error
}

func New(logger, messageLogger, callbackLogger *log.Logger, port int) *QuikService {
	return &QuikService{
		logger:         logger,
		messageLogger:  messageLogger,
		callbackLogger: callbackLogger,
		port:           port,
		queries:        make(chan query, 128),
		transId:        calculateStartTransId(),
	}
}

func (quik *QuikService) Events() Subscriber {
	return &quik.eventManager
}

func calculateStartTransId() int64 {
	var hour, min, sec = time.Now().Clock()
	return 60*(60*int64(hour)+int64(min)) + int64(sec)
}

func (quik *QuikService) executeQuery(ctx context.Context, command string,
	request, response interface{}) error {
	var query = query{
		command:  command,
		request:  request,
		response: response,
		done:     make(chan error),
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case quik.queries <- query:
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-query.done:
		return err
	}
}

func (quik *QuikService) DialAndServe(ctx context.Context) error {

	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	if quik.port == 0 {
		quik.port = 34130
	}

	mainConn, err := net.Dial("tcp", "localhost:"+strconv.Itoa(quik.port))
	if err != nil {
		return err
	}
	defer mainConn.Close()

	callbackConn, err := net.Dial("tcp", "localhost:"+strconv.Itoa(quik.port+1))
	if err != nil {
		return err
	}
	defer callbackConn.Close()

	var responses = make(chan string, 128)
	var callbacks = make(chan string, 128)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-callbacks:
			}
		}
	}()

	go func() {
		var err = readLines(ctx, quik.messageLogger, responses, mainConn)
		if err != nil {
			cancel()
			quik.logger.Println(err)
		}
	}()

	go func() {
		var err = handleCallbacks(quik.callbackLogger, callbackConn, &quik.eventManager)
		if err != nil {
			cancel()
			quik.logger.Println(err)
		}
	}()

	return handleQueries(ctx, quik.messageLogger, quik.queries, mainConn, responses)
}

func handleQueries(ctx context.Context,
	messageLogger *log.Logger,
	queries <-chan query,
	requests io.Writer,
	responses <-chan string) error {

	type RequestJson struct {
		Id          int64       `json:"id"`
		Command     string      `json:"cmd"`
		CreatedTime int64       `json:"t"`
		Data        interface{} `json:"data"`
	}

	type ResponseJson struct {
		Id          int64            `json:"id"`
		Command     string           `json:"cmd"`
		CreatedTime float64          `json:"t"`
		Data        *json.RawMessage `json:"data"`
		LuaError    string           `json:"lua_error"`
	}

	type nullable interface {
		Valid() bool
		SetValid(v bool)
	}

	var id = computeStartId()
	var queriesInProgress = make(map[int64]query)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case query := <-queries:
			id++
			var r = RequestJson{
				Id:          id,
				Command:     query.command,
				CreatedTime: timeToQuikTime(time.Now()),
				Data:        query.request,
			}
			b, err := json.Marshal(r)
			if err != nil {
				return err
			}
			var msg = string(b)
			messageLogger.Println(msg)
			_, err = requests.Write([]byte(msg + "\r\n"))
			if err != nil {
				return err
			}
			queriesInProgress[id] = query
		case incoming := <-responses:
			var responseJson ResponseJson
			var err = json.Unmarshal([]byte(incoming), &responseJson)
			if err != nil {
				return err
			}
			var query, ok = queriesInProgress[responseJson.Id]
			if !ok {
				return fmt.Errorf("Query not found %v", responseJson.Id)
			}
			delete(queriesInProgress, responseJson.Id)
			if responseJson.LuaError != "" {
				query.done <- errors.New(responseJson.LuaError)
				continue
			}
			if responseJson.Data != nil {
				err = json.Unmarshal(*responseJson.Data, query.response)
				if err != nil {
					query.done <- err
					continue
				}
				if nullable, ok := query.response.(nullable); ok {
					nullable.SetValid(true)
				}
			}
			close(query.done)
		}
	}
}

func computeStartId() int64 {
	return 1
}

func timeToQuikTime(time time.Time) int64 {
	return time.UnixNano() / 1000
}

func readLines(ctx context.Context,
	messageLogger *log.Logger,
	dst chan<- string,
	src io.Reader) error {
	reader := bufio.NewReader(transform.NewReader(src, charmap.Windows1251.NewDecoder()))
	for {
		incoming, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		if messageLogger != nil {
			messageLogger.Println(incoming)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case dst <- incoming:
		}
	}
}

const (
	EventNameOnConnected    = "OnConnected"
	EventNameOnDisconnected = "OnDisconnected"
	EventNameOnTrade        = "OnTrade"
	EventNameNewCandle      = "NewCandle"
)

func handleCallbacks(
	logger *log.Logger,
	src io.Reader,
	publisher Publisher) error {

	type ResponseJson struct {
		Command     string           `json:"cmd"`
		CreatedTime float64          `json:"t"`
		Data        *json.RawMessage `json:"data"`
		LuaError    string           `json:"lua_error"`
	}

	reader := bufio.NewReader(transform.NewReader(src, charmap.Windows1251.NewDecoder()))
	for {
		incoming, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		var responseJson ResponseJson
		err = json.Unmarshal([]byte(incoming), &responseJson)
		if err != nil {
			logger.Println(err, incoming)
			continue
		}
		if responseJson.LuaError != "" {
			logger.Println(responseJson.LuaError)
			continue
		}
		var msg interface{} = nil
		switch responseJson.Command {
		case EventNameOnConnected:
			msg = &ConnectedEventData{}
		case EventNameOnDisconnected:
			msg = &DisconnectedEventData{}
		case EventNameOnTrade:
			msg = &TradeEventData{}
		case EventNameNewCandle:
			msg = &NewCandleEventData{}
		}
		if msg != nil {
			if responseJson.Data != nil {
				err = json.Unmarshal(*responseJson.Data, msg)
				if err != nil {
					logger.Println(err, incoming)
					continue
				}
			}
			logger.Println(incoming)
			logger.Printf("%+v\n", msg)
			publisher.Publish(msg)
		}
	}
}
