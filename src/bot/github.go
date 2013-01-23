package bot

// A bot that listens for github commit notifications and posts them to channels.
// TODO: Listen for other events (e.g., new issues, pull requests, comments, etc).

import (
	"bytes"
	"connection"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"text/template"
)

const (
	addr = "localhost:9898"
)

// TODO: configuration for bots should probably be in config files
var config = struct {
	// repo -> channels to notify
	Notifications map[string][]string
	// repo -> default project (e.g. bkad/prat)
	Issues map[string]string
}{
	Notifications: map[string][]string{
		"oochat":  {"general", "oochat"},
		"barkeep": {"barkeep"},
		"pratbot": {"pratbot", "bot-test"},
	},
	Issues: map[string]string{
		"general": "bkad/oochat",
		"pratbot": "cespare/pratbot",
		"barkeep": "ooyala/barkeep",
	},
}
var chans = make(map[string]struct{})
var templ *template.Template

func init() {
	// Get all unique channels
	for _, cs := range config.Notifications {
		for _, c := range cs {
			chans[c] = struct{}{}
		}
	}
	for c, _ := range config.Issues {
		chans[c] = struct{}{}
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
**[GithubBot]** [{{.Author.Name}}](https://github.com/{{.Author.Username}}) authored [{{.Id | shortenSha}}]({{.Url}}) in [{{$repo.Name}}]({{$repo.Url}}): "{{.Message | shortenMessage}}"
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
			log.Println("GithubBot warning: couldn't parse payload:", payload)
			return
		}
		var buf bytes.Buffer
		if err := templ.Execute(&buf, &notification); err != nil {
			log.Println("GithubBot warning: couldn't construct message", err)
			return
		}
		message := strings.TrimSpace(buf.String())
		for _, c := range config.Notifications[notification.Repository.Name] {
			conn.SendMessage(c, message)
		}
	}
}

type Github struct {
	conn *connection.Conn
	ui   *UserInfo
}

func NewGithub(conn *connection.Conn, ui *UserInfo) Bot {
	return &Github{conn, ui}
}

func (b *Github) Send(channel, msg string) {
	b.conn.SendMessage(channel, "**[GithubBot]** "+msg)
}

func (b *Github) SendIssueError(channel, msg string) {
	msg = "**error:** " + msg + " (hint: try '!issue <commit>' or '!issue <user>/<repo> <commit>')"
	b.Send(channel, msg)
}

type issueResponse struct {
	Url   string `json:"html_url"`
	State string
	Title string
}

func (b *Github) IssueLookup(channel, msg string) {
	parts := strings.Split(msg, " ")
	repo := config.Issues[channel]
	var issue string
	switch len(parts) {
	case 1:
		issue = parts[0]
	case 2:
		repo = parts[0]
		issue = parts[1]
	default:
		b.SendIssueError(channel, "bad input.")
		return
	}
	repoParts := strings.SplitN(repo, "/", 2)
	if len(repoParts) != 2 {
		b.SendIssueError(channel, "Bad repo (should be owner/repo): "+repo)
		return
	}
	owner, repo := repoParts[0], repoParts[1]
	issueNumber, err := strconv.Atoi(issue)
	if err != nil {
		b.SendIssueError(channel, "Bad issue (should be a number): "+issue)
		return
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d", owner, repo, issueNumber)
	resp, err := http.Get(url)
	if err != nil {
		b.SendIssueError(channel, "Error fetching issue info.")
		return
	}
	switch resp.StatusCode {
	case 200:
	case 404:
		b.Send(channel, "No such issue.")
		return
	default:
		b.SendIssueError(channel, "Error fetching issue info.")
		return
	}
	r := &issueResponse{}
	var buf bytes.Buffer
	io.Copy(&buf, resp.Body)
	err = json.Unmarshal(buf.Bytes(), r)
	if err != nil {
		b.SendIssueError(channel, "Error decoding issue info.")
		return
	}
	responseMsg := fmt.Sprintf(`[Issue #%d in %s/%s:](%s) %s **[%s]**`,
		issueNumber, owner, repo, r.Url, r.Title, r.State)
	b.Send(channel, responseMsg)
}

func (b *Github) Handle(e *Event) {
	switch e.Type {
	case EventConnect:
		// We don't really need to join these channels, but whatever.
		for c, _ := range chans {
			b.conn.Join(c)
		}

		// Start server
		mux := http.NewServeMux()
		mux.HandleFunc("/", NotificationHandler(b.conn))
		go http.ListenAndServe(addr, mux)
	case EventPublishMessage:
		m := e.Payload.(PublishMessage)
		// Ignore our own message.
		if m.Data.User.Email == b.ui.User.Email {
			return
		}

		// Respond to issue requests
		prefix := "!issue"
		if strings.HasPrefix(m.Data.Message, prefix) {
			for channel := range config.Issues {
				if channel == m.Data.Channel {
					b.IssueLookup(channel, strings.TrimSpace(m.Data.Message[len(prefix):]))
				}
			}
		}
	}
}
