package ws

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/cfoust/cy/pkg/util"

	"nhooyr.io/websocket"
)

type Client interface {
	Ctx() context.Context
	Send(data []byte) error
	Receive() <-chan []byte
	Close(code websocket.StatusCode, reason string) error
}

type WSClient struct {
	util.Session
	Conn *websocket.Conn
}

const (
	WRITE_TIMEOUT = 1 * time.Second
)

func (c *WSClient) Send(data []byte) error {
	ctx, cancel := context.WithTimeout(c.Ctx(), WRITE_TIMEOUT)
	defer cancel()
	return c.Conn.Write(ctx, websocket.MessageBinary, data)
}

func (c *WSClient) Receive() <-chan []byte {
	ctx := c.Ctx()
	out := make(chan []byte)
	go func() {
		for {
			if ctx.Err() != nil {
				return
			}

			typ, message, err := c.Conn.Read(ctx)
			if err != nil {
				return
			}

			if typ != websocket.MessageBinary {
				continue
			}

			out <- message
		}
	}()

	return out
}

func (c *WSClient) Close(code websocket.StatusCode, reason string) error {
	return c.Close(code, reason)
}

var _ Client = (*WSClient)(nil)

func Connect(ctx context.Context, socketPath string) (Client, error) {
	// https://gist.github.com/teknoraver/5ffacb8757330715bcbcc90e6d46ac74
	httpClient := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}

	options := websocket.DialOptions{
		HTTPClient: &httpClient,
	}
	c, _, err := websocket.Dial(ctx, "http://unix/", &options)
	if err != nil {
		return nil, err
	}

	client := WSClient{
		Session: util.NewSession(ctx),
		Conn:    c,
	}

	go func() {
		<-client.Ctx().Done()
		c.Close(websocket.StatusNormalClosure, "")
	}()

	return &client, nil
}
