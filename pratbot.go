package main

import (
	"flag"
	"fmt"
	"log"
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
	tls        = flag.Bool("tls", true, "Connect via TLS")
	port       = flag.Int("port", 0, "Port (defaults to 80/443)")
	botsString = flag.String("bots", "", "Comma-separated list of bots to initialize")

	addrString string
)

type newBotFunc func(*connection.Conn) bot.Bot

var (
	botNameToFunc = map[string]newBotFunc{
		"echo":   bot.NewEcho,
		"commit": bot.NewCommit,
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
		if *tls {
			*port = 443
		} else {
			*port = 80
		}
	}

	proto := "ws"
	if *tls {
		proto = "wss"
	}
	addrString = fmt.Sprintf("%s://%s:%d", proto, *server, *port)

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
	conn, err := connection.Connect(addrString, *apiKey, *secret)
	if err != nil {
		log.Fatal(err)
	}

	// Register bots
	for _, f := range bots {
		disp.Register(f(conn))
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
