package main

import (
	"log"
	"flag"
	"fmt"
	"os"

	"connection"
)

var (
	server = flag.String("server", "", "Prat server")
	apiKey = flag.String("apikey", "", "Prat API key")
	secret = flag.String("secret", "", "Prat API secret")
	tls = flag.Bool("tls", true, "Connect via TLS")
	port = flag.Int("port", 0, "Port (defaults to 80/443)")

	addrString string
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
}

func main() {
	conn, err := connection.Connect(addrString, *apiKey, *secret)
	if err != nil {
		log.Fatal(err)
	}
	if conn.SendMessage("general", "Hello from bot!") != nil {
		log.Fatal(err)
	}
}
