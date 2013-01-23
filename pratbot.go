package main

import (
	"authutil"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"bot"
	"connection"
	"dispatcher"
)

var (
	server     = flag.String("server", "", "Prat server")
	apiKey     = flag.String("apikey", "", "Prat API key")
	secret     = flag.String("secret", "", "Prat API secret")
	useTls     = flag.Bool("tls", true, "Connect via TLS")
	port       = flag.Int("port", 0, "Port (defaults to 80/443)")
	botsString = flag.String("bots", "", "Comma-separated list of bots to initialize")

	wsAddr   string
	httpAddr string
)

type newBotFunc func(*connection.Conn, *bot.UserInfo) bot.Bot

var (
	botNameToFunc = map[string]newBotFunc{
		"echo":   bot.NewEcho,
		"github": bot.NewGithub,
	}
	bots = make(map[string]newBotFunc)
	disp = dispatcher.New()
)

func init() {
	flag.Parse()

	for _, f := range []string{*server, *apiKey, *secret} {
		if f == "" {
			flag.Usage()
			os.Exit(-1)
		}
	}

	if *port == 0 {
		if *useTls {
			*port = 443
		} else {
			*port = 80
		}
	}

	proto := ""
	if *useTls {
		proto = "s"
	}
	wsAddr = fmt.Sprintf("ws%s://%s:%d", proto, *server, *port)
	httpAddr = fmt.Sprintf("http%s://%s:%d", proto, *server, *port)

	botList := strings.Split(*botsString, ",")
	for _, bs := range botList {
		if bs == "" {
			continue
		}
		f, ok := botNameToFunc[bs]
		if !ok {
			log.Fatalln("Unrecognized bot:", bs)
		}
		bots[bs] = f
	}
	if len(bots) == 0 {
		log.Fatalln("Must specify one or more bots to run.")
	}
}

func main() {
	// Connect
	conn, err := connection.Connect(wsAddr, *apiKey, *secret)
	if err != nil {
		log.Fatal(err)
	}

	// Get info about ourself.
	addr := httpAddr + authutil.SignRequest("/api/whoami", *apiKey, *secret)
	// For some reason I can't verify the cert on pratchat.com :\
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: transport}
	response, err := client.Get(addr)
	if err != nil {
		log.Fatal("Error fetching user info: " + err.Error())
	}
	var buf bytes.Buffer
	io.Copy(&buf, response.Body)
	userInfo := &bot.UserInfo{}
	if err := json.Unmarshal(buf.Bytes(), userInfo); err != nil {
		log.Fatal("Error getting user info: " + err.Error())
	}

	// Leave all current channels.
	for _, channel := range userInfo.User.Channels {
		conn.Leave(channel)
	}

	// Register bots
	for _, f := range bots {
		disp.Register(f(conn, userInfo))
	}

	log.Println("Bots started.")

	// Send 'connected' message
	connectedMsg := &bot.Event{
		Type: bot.EventConnect,
	}
	disp.Send(connectedMsg)

	// Loop, receiving messages, and send them through the dispatcher
	for {
		msg := <-conn.In
		disp.SendRaw(msg)
	}
}
