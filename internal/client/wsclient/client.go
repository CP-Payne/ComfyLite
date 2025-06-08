package wsclient

import (
	"fmt"
	"net/url"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type Client struct {
	logger *zap.Logger
	host   string
	port   string
	conn   *websocket.Conn
}

func New(logger *zap.Logger, host, port string) *Client {
	return &Client{
		logger: logger,
		host:   host,
		port:   port,
	}
}

func (wc *Client) Connect() (string, error) {

	clientID := uuid.New()

	url := url.URL{
		Scheme:   "ws",
		Host:     fmt.Sprintf("%s:%s", wc.host, wc.port),
		Path:     "/ws",
		RawQuery: fmt.Sprintf("clientId=%s", clientID.String()),
	}

	conn, resp, err := websocket.DefaultDialer.Dial(url.String(), nil)
	if err != nil {
		wc.logger.Error("Client: Dial error", zap.Any("HTTPResponse", resp), zap.Error(err))
		return "", err
	}

	wc.conn = conn

	return clientID.String(), nil
}

func (wc *Client) Listen(handler func(int, []byte), wg *sync.WaitGroup) {
	defer wc.conn.Close()

	wg.Done()
	for {
		messageType, message, err := wc.conn.ReadMessage()
		if err != nil {
			wc.logger.Error("Websocket read error", zap.Error(err))
			break
		}

		go handler(messageType, message)
	}
}
