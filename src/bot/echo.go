package bot

import (
	"connection"
	"strings"
)

var channels = []string{"bot-test"}

type Echo struct {
	conn *connection.Conn
	ui   *UserInfo
}

func NewEcho(conn *connection.Conn, ui *UserInfo) Bot {
	return &Echo{conn, ui}
}

func (b *Echo) Handle(e *Event) {
	switch e.Type {
	case EventConnect:
		for _, c := range channels {
			b.conn.Join(c)
		}
	case EventPublishMessage:
		m := e.Payload.(PublishMessage)
		// Ignore our own message.
		if m.Data.User.Email == b.ui.User.Email {
			return
		}

		msg := m.Data.Message
		channel := m.Data.Channel
		newMessage := "**" + strings.ToUpper(msg) + "**"
		b.conn.SendMessage(channel, newMessage)
	}
}
