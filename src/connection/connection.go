package connection

import (
	"code.google.com/p/go.net/websocket"
	"crypto/tls"
	"encoding/json"
	"log"
	"time"

	"authutil"
)

var (
	PingFrequency = 30 * time.Second
	// TODO: exponential backoff
	ReconnectFrequency = 10 * time.Second
)

func (c *Conn) receive() {
	for {
		var msg string
		if err := websocket.Message.Receive(c.ws, &msg); err != nil {
			log.Println("Error receiving message:", err)
			// TODO:
			// * Put the sleeping in the reconnection function itself
			// * Also do this when there's a send failure
			// * Don't try to send or receive at all when we're disconnected
			for {
				err := c.reconnect()
				if err == nil {
					log.Println("Reconnection successful.")
					break
				}
				log.Println("Reconnection failed:", err)
				time.Sleep(ReconnectFrequency)
			}
		}
		c.In <- msg
	}
}

func (c *Conn) ping() {
	ping := &Message{
		Action: "ping",
		Data:   map[string]string{"message": "PING"},
	}
	j, err := json.Marshal(ping)
	if err != nil {
		log.Fatal(err)
	}
	// Send a heartbeat ping every N seconds.
	ticker := time.NewTicker(PingFrequency)
	for _ = range ticker.C {
		c.out <- string(j)
	}
}

func (c *Conn) send() {
	for msg := range c.out {
		if err := websocket.Message.Send(c.ws, msg); err != nil {
			log.Println("Error sending message:", err)
		}
	}
}

type credentials struct {
	addrString, apiKey, secret string
}

type Conn struct {
	ws *websocket.Conn
	// Messages come out here
	In    chan string
	out   chan string
	creds *credentials
}

type Message struct {
	Action string            `json:"action"`
	Data   map[string]string `json:"data"`
}

func (c *Conn) sendJsonData(d interface{}) error {
	j, err := json.Marshal(d)
	if err != nil {
		return err
	}
	c.out <- string(j)
	return nil
}

func (c *Conn) SendMessage(channel, msg string) error {
	data := map[string]string{"channel": channel, "message": msg}
	m := &Message{
		Action: "publish_message",
		Data:   data,
	}
	return c.sendJsonData(m)
}

func (c *Conn) Join(channel string) error {
	data := map[string]string{"channel": channel}
	m := &Message{
		Action: "join_channel",
		Data:   data,
	}
	return c.sendJsonData(m)
}

func (c *Conn) Leave(channel string) error {
	data := map[string]string{"channel": channel}
	m := &Message{
		Action: "leave_channel",
		Data:   data,
	}
	return c.sendJsonData(m)
}

func (c *Conn) connect() error {
	connectionString := c.creds.addrString + authutil.SignRequest("/eventhub", c.creds.apiKey, c.creds.secret)
	config, err := websocket.NewConfig(connectionString, "http://localhost")
	if err != nil {
		return err
	}
	// Ignore certs for now
	config.TlsConfig = &tls.Config{InsecureSkipVerify: true}
	ws, err := websocket.DialConfig(config)
	if err != nil {
		return err
	}
	c.ws = ws
	return nil
}

func (c *Conn) reconnect() error {
	if err := c.connect(); err != nil {
		return err
	}
	// TODO: shouldn't have a hard-coded channel for this, but instead a configurable status channel.
	c.SendMessage("pratbot", "Pratbot reconnected.")
	return nil
}

func Connect(addrString, apiKey, secret string) (*Conn, error) {
	conn := &Conn{creds: &credentials{addrString, apiKey, secret}}
	if err := conn.connect(); err != nil {
		return nil, err
	}
	conn.In = make(chan string)
	conn.out = make(chan string)

	// Start goroutines
	go conn.receive()
	go conn.ping()
	go conn.send()

	return conn, nil
}
