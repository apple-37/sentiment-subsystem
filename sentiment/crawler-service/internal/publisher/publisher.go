package publisher

import (
	"encoding/json"

	"github.com/streadway/amqp"
)

type RabbitMQPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   amqp.Queue
}

func NewRabbitMQPublisher(mqURL, queueName string) (*RabbitMQPublisher, error) {
	conn, err := amqp.Dial(mqURL)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	q, err := ch.QueueDeclare(queueName, true, false, false, false, nil)
	return &RabbitMQPublisher{conn, ch, q}, err
}

// Publish 发送事件到 MQ
func (p *RabbitMQPublisher) Publish(event any) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.channel.Publish(
		"",           // exchange
		p.queue.Name, // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
}

// Close 释放 MQ 资源
func (p *RabbitMQPublisher) Close() {
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
}