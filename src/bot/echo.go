package bot

import (
	"connection"
	"strings"
)

var channels = []string{"bot-test"}

type Echo struct {
	conn *connection.Conn
}

func NewEcho(conn *connection.Conn) Bot {
	return &Echo{
		conn: conn,
	}
}

func (b *Echo) Handle(e *Event) {
	switch e.Type {
	case EventConnect:
		for _, c := range channels {
			b.conn.Join(c)
		}
	case EventPublishMessage:
		m := e.Payload.(PublishMessage)

		// Ignore our own messages. Hard-code username until we make an API to retrieve this info.
		if m.Data.Username == "foo" {
			return
		}

		msg := m.Data.Message
		channel := m.Data.Channel
		newMessage := "**" + strings.ToUpper(msg) + "**"
		b.conn.SendMessage(channel, newMessage)
	}
}
