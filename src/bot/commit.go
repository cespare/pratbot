package bot

// A bot that listens for github commit notifications and posts them to channels.

import (
	"bytes"
	"connection"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"text/template"
)

const (
	addr = "localhost:9898"
)

// TODO: configuration for bots should probably be in config files
var repoToChans = map[string][]string{
	"oochat":  {"general", "oochat"},
	"barkeep": {"barkeep"},
	"pratbot": {"pratbot", "bot-test"},
}
var chans = make(map[string]struct{})
var templ *template.Template

func init() {
	// Get all unique channels
	for _, cs := range repoToChans {
		for _, c := range cs {
			chans[c] = struct{}{}
		}
	}

	// Set up template
	funcMap := template.FuncMap{
		"shortenSha":     shortenSha,
		"shortenMessage": shortenMessage,
	}
	var err error
	templ, err = template.New("message").Funcs(funcMap).Parse(messageTemplate)
	if err != nil {
		log.Fatal(err)
	}
}

// Set up server that gets github post-receive hook POST requests.

// Only the fields we care about
type GithubNotification struct {
	Repository struct {
		Name string
		Url  string
	}
	Commits []struct {
		Id      string
		Message string
		Url     string
		Author  struct {
			Name     string
			Username string
		}
	}
}

func shortenMessage(msg string) string {
	subject := strings.SplitN(msg, "\n", 2)[0]
	if len(subject) > 80 {
		return subject[:77] + "..."
	}
	return subject
}

func shortenSha(sha string) string {
	return sha[:8]
}

var messageTemplate = `
{{$repo := .Repository}}
{{range .Commits}}
**[CommitBot]** [{{.Author.Name}}](https://github.com/{{.Author.Username}}) authored [{{.Id | shortenSha}}]({{.Url}}) in [{{$repo.Name}}]({{$repo.Url}}): "{{.Message | shortenMessage}}"
{{end}}
`

func NotificationHandler(conn *connection.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		payload := r.Form["payload"]
		if len(payload) < 1 || payload[0] == "" {
			return
		}
		var notification GithubNotification
		if err := json.Unmarshal([]byte(payload[0]), &notification); err != nil {
			log.Println(err)
			log.Println("CommitBot warning: couldn't parse payload:", payload)
			return
		}
		var buf bytes.Buffer
		if err := templ.Execute(&buf, &notification); err != nil {
			log.Println("CommitBot warning: couldn't construct message", err)
			return
		}
		message := strings.TrimSpace(buf.String())
		for _, c := range repoToChans[notification.Repository.Name] {
			conn.SendMessage(c, message)
		}
	}
}

type Commit struct {
	conn *connection.Conn
}

func NewCommit(conn *connection.Conn) Bot {
	return &Commit{conn}
}

func (b *Commit) Handle(e *Event) {
	switch e.Type {
	case EventConnect:
		b.conn.Leave("general") // quick hack until there's an API for current channels
		// We don't really need to join these channels, but whatever.
		for c, _ := range chans {
			b.conn.Join(c)
		}

		// Start server
		mux := http.NewServeMux()
		mux.HandleFunc("/", NotificationHandler(b.conn))
		go http.ListenAndServe(addr, mux)
	}
}
