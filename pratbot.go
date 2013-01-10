package main

import (
	"bytes"
	"code.google.com/p/go.net/websocket"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	server = "pratchat.com:443"
)

var (
	apikey = flag.String("apikey", "", "Prat API key")
	secret = flag.String("secret", "", "Prat API secret")
	ws     *websocket.Conn
)

func init() {
	flag.Parse()

	if *apikey == "" || *secret == "" {
		flag.Usage()
		os.Exit(-1)
	}
}

type ByPair [][]string

func (p ByPair) Len() int           { return len(p) }
func (p ByPair) Less(i, j int) bool { return p[i][0] < p[j][0] }
func (p ByPair) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func prepareQueryString(params map[string]string, exclude []string) string {
	var result [][]string
	for k, v := range params {
		excluded := false
		for _, e := range exclude {
			if k == e {
				excluded = true
				break
			}
		}
		if !excluded {
			result = append(result, []string{k, v})
		}
	}
	sort.Sort(ByPair(result))
	var pairs []string
	for _, p := range result {
		pairs = append(pairs, strings.Join(p, "="))
	}
	return strings.Join(pairs, "")
}

func Signature(secret, method, path, body string, params map[string]string) string {
	exclude := []string{"signature"}
	signature := secret + strings.ToUpper(method) + path + prepareQueryString(params, exclude) + body
	h := sha256.New()
	h.Write([]byte(signature))
	var buf bytes.Buffer
	enc := base64.NewEncoder(base64.URLEncoding, &buf)
	enc.Write(h.Sum(nil))
	enc.Close()
	return buf.String()[:43]
}

func ConnectionString() string {
	expires := int(time.Now().Unix() + 300)
	params := map[string]string{"api_key": *apikey, "expires": strconv.Itoa(expires)}
	signature := Signature(*secret, "GET", "/eventhub", "", params)
	params["signature"] = signature
	var kv []string
	for k, v := range params {
		kv = append(kv, k+"="+v)
	}
	return fmt.Sprintf("wss://%s/eventhub?%s", server, strings.Join(kv, "&"))
}

type Message struct {
	Action string            `json:"action"`
	Data   map[string]string `json:"data"`
}

func SendMessage(channel, msg string) error {
	data := map[string]string{"channel": channel, "message": msg}
	m := &Message{
		Action: "publish_message",
		Data:   data,
	}
	j, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if _, err := ws.Write(j); err != nil {
		return err
	}
	return nil
}

func main() {
	origin := "http://localhost/"
	var err error
	config, err := websocket.NewConfig(ConnectionString(), origin)
	if err != nil {
		log.Fatal(err)
	}
	// Ignore certs for now
	config.TlsConfig = &tls.Config{InsecureSkipVerify: true}
	ws, err = websocket.DialConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connected.")
	if SendMessage("general", "Hello from bot!") != nil {
		log.Fatal(err)
	}
}
