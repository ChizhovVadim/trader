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
	eventManager   *EventManager
}

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

type CallbackJson struct {
	Command     string           `json:"cmd"`
	CreatedTime float64          `json:"t"`
	Data        *json.RawMessage `json:"data"`
	LuaError    string           `json:"lua_error"`
}

type query struct {
	command  string
	request  interface{}
	response chan ResponseJson
}

func New(logger, messageLogger, callbackLogger *log.Logger, port int) *QuikService {
	return &QuikService{
		logger:         logger,
		messageLogger:  messageLogger,
		callbackLogger: callbackLogger,
		port:           port,
		queries:        make(chan query),
		transId:        calculateStartTransId(),
		eventManager:   &EventManager{},
	}
}

func (quik *QuikService) Events() Subscriber {
	return quik.eventManager
}

func calculateStartTransId() int64 {
	var hour, min, sec = time.Now().Clock()
	return 60*(60*int64(hour)+int64(min)) + int64(sec)
}

func (quik *QuikService) ExecuteQuery(
	ctx context.Context,
	command string,
	request, response interface{},
) error {
	var query = query{
		command:  command,
		request:  request,
		response: make(chan ResponseJson, 1),
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case quik.queries <- query:
	}

	type nullable interface {
		Valid() bool
		SetValid(v bool)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case responseJson := <-query.response:
		if responseJson.LuaError != "" {
			return errors.New(responseJson.LuaError)
		}
		if responseJson.Data != nil && response != nil {
			var err = json.Unmarshal(*responseJson.Data, response)
			if err != nil {
				return err
			}
			if nullable, ok := response.(nullable); ok {
				nullable.SetValid(true)
			}
		}
		return nil
	}
}

func (quik *QuikService) DialAndServe(ctx context.Context) error {
	return runQuik(ctx, quik.logger, quik.messageLogger, quik.callbackLogger,
		quik.port, quik.eventManager, quik.queries)
}

func runQuik(
	ctx context.Context,
	logger, messageLogger, callbackLogger *log.Logger,
	port int,
	publisher Publisher,
	queries <-chan query,
) error {
	if port == 0 {
		port = 34130
	}

	mainConn, err := net.Dial("tcp", "localhost:"+strconv.Itoa(port))
	if err != nil {
		return err
	}
	defer mainConn.Close()

	callbackConn, err := net.Dial("tcp", "localhost:"+strconv.Itoa(port+1))
	if err != nil {
		return err
	}
	defer callbackConn.Close()

	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()

	var responses = make(chan string)

	go func() {
		var err = handleResponses(ctx, messageLogger, responses, mainConn)
		if err != nil {
			cancel()
			logger.Println(err)
		}
	}()
	go func() {
		var err = handleCallbacks(ctx, logger, callbackLogger, callbackConn, publisher)
		if err != nil {
			cancel()
			logger.Println(err)
		}
	}()
	return handleQueries(ctx, logger, messageLogger, queries, mainConn, responses)
}

func handleQueries(
	ctx context.Context,
	logger, messageLogger *log.Logger,
	queries <-chan query,
	requests io.Writer,
	responses <-chan string,
) error {
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
			if responseJson.LuaError != "" {
				logger.Println(responseJson.LuaError)
			}
			var query, ok = queriesInProgress[responseJson.Id]
			if !ok {
				return fmt.Errorf("query not found %v", responseJson.Id)
			}
			delete(queriesInProgress, responseJson.Id)
			query.response <- responseJson
		}
	}
}

func computeStartId() int64 {
	return 1
}

func timeToQuikTime(time time.Time) int64 {
	return time.UnixNano() / 1000
}

func handleResponses(
	ctx context.Context,
	messageLogger *log.Logger,
	responses chan<- string,
	r io.Reader,
) error {
	reader := bufio.NewReader(transform.NewReader(r, charmap.Windows1251.NewDecoder()))
	for {
		incoming, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		messageLogger.Println(incoming)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case responses <- incoming:
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
	ctx context.Context,
	logger, callbackLogger *log.Logger,
	r io.Reader,
	publisher Publisher,
) error {
	reader := bufio.NewReader(transform.NewReader(r, charmap.Windows1251.NewDecoder()))
	for {
		incoming, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		var callbackJson CallbackJson
		err = json.Unmarshal([]byte(incoming), &callbackJson)
		if err != nil {
			return err
		}
		//TODO публиковать CallbackJson, а там уже обработчики или цепочка ответственности пусть парсят остальное, если им нужно
		if callbackJson.LuaError != "" {
			logger.Println(callbackJson.LuaError)
			callbackLogger.Println(incoming)
			continue
		}

		var msg interface{} = nil
		switch callbackJson.Command {
		case EventNameOnConnected:
			//msg = &ConnectedEventData{}
		case EventNameOnDisconnected:
			//msg = &DisconnectedEventData{}
		case EventNameOnTrade:
			msg = &TradeEventData{}
		case EventNameNewCandle:
			msg = &NewCandleEventData{}
		}

		// чтобы не засорять лог
		if cmdName := callbackJson.Command; !(cmdName == "OnParam" ||
			cmdName == "OnFuturesLimitChange" ||
			cmdName == "OnFuturesClientHolding") {
			callbackLogger.Println(incoming)
		}

		if msg != nil && callbackJson.Data != nil {
			err = json.Unmarshal(*callbackJson.Data, msg)
			if err != nil {
				logger.Println(err, incoming)
				continue
			}
			callbackLogger.Printf("%+v\n", msg)
			publisher.Publish(msg)
		}
	}
}
