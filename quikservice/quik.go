package quikservice

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

type QuikService struct {
	logger       *log.Logger
	port         int
	id           int64
	mu           sync.Mutex
	mainConn     net.Conn
	reader       *bufio.Reader
	writer       *transform.Writer
	callbackConn net.Conn
}

func New(
	logger *log.Logger,
	port int,
	id int64,
) *QuikService {
	return &QuikService{
		logger: logger,
		port:   port,
		id:     id,
	}
}

func (quik *QuikService) Init(
	ctx context.Context,
	callbackHandler func(context.Context, CallbackJson),
) error {
	mainConn, err := dial(quik.port)
	if err != nil {
		return err
	}
	quik.mainConn = mainConn

	callbackConn, err := dial(quik.port + 1)
	if err != nil {
		return err
	}
	quik.callbackConn = callbackConn

	var quikCharmap = charmap.Windows1251
	quik.reader = bufio.NewReader(transform.NewReader(quik.mainConn, quikCharmap.NewDecoder()))
	quik.writer = transform.NewWriter(quik.mainConn, quikCharmap.NewEncoder())

	// эта горутина завершатся, тк defer quik.Close() закроет callback connection.
	// даже если не хотим обрабатывать callbacks, то все равно нужно читать сообщения.
	go func() {
		quik.handleCallbacks(ctx, callbackHandler)
	}()
	return nil
}

func (quik *QuikService) Close() error {
	var mainConnErr, callbackConnErr error
	if quik.mainConn != nil {
		mainConnErr = quik.mainConn.Close()
	}
	if quik.callbackConn != nil {
		callbackConnErr = quik.callbackConn.Close()
	}
	return errors.Join(mainConnErr, callbackConnErr)
}

func dial(port int) (net.Conn, error) {
	return net.Dial("tcp", "localhost:"+strconv.Itoa(port))
}

func timeToQuikTime(time time.Time) int64 {
	return time.UnixNano() / 1000
}

func (quik *QuikService) MakeQuery(cmd string, data any) (ResponseJson, error) {
	quik.mu.Lock()
	defer quik.mu.Unlock()

	var request = RequestJson{
		Id:          quik.id,
		Command:     cmd,
		CreatedTime: timeToQuikTime(time.Now()),
		Data:        data,
	}
	quik.id += 1

	b, err := json.Marshal(request)
	if err != nil {
		return ResponseJson{}, err
	}
	_, err = quik.writer.Write(b)
	if err != nil {
		return ResponseJson{}, err
	}
	_, err = quik.writer.Write([]byte("\r\n"))
	if err != nil {
		return ResponseJson{}, err
	}

	if quik.logger != nil {
		quik.logger.Println(string(b))
	}

	incoming, err := quik.reader.ReadString('\n')
	if err != nil {
		return ResponseJson{}, err
	}
	if quik.logger != nil && len(incoming) <= 2_048 {
		quik.logger.Println(incoming)
	}
	var response ResponseJson
	err = json.Unmarshal([]byte(incoming), &response)
	if err != nil {
		return ResponseJson{}, err
	}
	if response.LuaError != "" {
		return ResponseJson{}, fmt.Errorf("lua error: %v", response.LuaError)
	}
	return response, nil
}

func (quik *QuikService) handleCallbacks(
	ctx context.Context,
	callbackHandler func(context.Context, CallbackJson),
) error {
	reader := bufio.NewReader(transform.NewReader(quik.callbackConn, charmap.Windows1251.NewDecoder()))
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
		if callbackHandler != nil {
			callbackHandler(ctx, callbackJson)
		}
	}
}

type RequestJson struct {
	Id          int64  `json:"id"`
	Command     string `json:"cmd"`
	CreatedTime int64  `json:"t"`
	Data        any    `json:"data"`
}

type ResponseJson struct {
	Id          int64   `json:"id"`
	Command     string  `json:"cmd"`
	CreatedTime float64 `json:"t"`
	Data        any     `json:"data"`
	LuaError    string  `json:"lua_error"`
}

type CallbackJson struct {
	Command     string  `json:"cmd"`
	CreatedTime float64 `json:"t"`
	Data        any     `json:"data"`
	LuaError    string  `json:"lua_error"`
}
