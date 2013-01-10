package dispatcher

import (
	"bot"
	"encoding/json"
	"log"
)

type Dispatcher struct {
	bots []bot.Bot
}

func New() *Dispatcher {
	return &Dispatcher{}
}

func (d *Dispatcher) Register(b bot.Bot) {
	d.bots = append(d.bots, b)
}

func (d *Dispatcher) Send(e *bot.Event) {
	for _, b := range d.bots {
		b.Handle(e)
	}
}

func (d *Dispatcher) SendRaw(msg string) {
	typ := bot.MessageType{}
	if err := json.Unmarshal([]byte(msg), &typ); err != nil {
		log.Println("Warning: received bad message:", err)
		return
	}
	event := &bot.Event{}
	switch typ.Type {
	case "publish_message":
		m := new(bot.PublishMessage)
		if err := json.Unmarshal([]byte(msg), m); err != nil {
			log.Println("Warning: bad publish message:", err)
			return
		}
		event.Type = bot.EventPublishMessage
		event.Payload = *m
	default:
		log.Println("Received unhandled message type:", typ.Type)
		return
	}
	d.Send(event)
}
