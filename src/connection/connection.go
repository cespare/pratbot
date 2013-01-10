package connection

import (
	"encoding/json"
	"code.google.com/p/go.net/websocket"
	"crypto/tls"

	"authutil"
)

type Conn struct {
	ws *websocket.Conn
}

type Message struct {
	Action string            `json:"action"`
	Data   map[string]string `json:"data"`
}

func (c *Conn) SendMessage(channel, msg string) error {
	data := map[string]string{"channel": channel, "message": msg}
	m := &Message{
		Action: "publish_message",
		Data:   data,
	}
	j, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if _, err := c.ws.Write(j); err != nil {
		return err
	}
	return nil
}

func Connect(addrString, apiKey, secret string) (*Conn, error) {
	origin := "http://localhost/"
	config, err := websocket.NewConfig(authutil.ConnectionString(addrString, apiKey, secret), origin)
	if err != nil {
		return nil, err
	}
	// Ignore certs for now
	config.TlsConfig = &tls.Config{InsecureSkipVerify: true}
	ws, err := websocket.DialConfig(config)
	if err != nil {
		return nil, err
	}
	return &Conn{ws}, nil
}
