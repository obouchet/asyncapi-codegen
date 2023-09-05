package controllers

import (
	"context"
	"errors"

	"github.com/lerenn/asyncapi-codegen/pkg/broker"
	"github.com/lerenn/asyncapi-codegen/pkg/log"
	"github.com/nats-io/nats.go"
)

// NATS is the NATS implementation for asyncapi-codegen
type NATS struct {
	connection *nats.Conn
	logger     log.Interface
	queueName  string
}

// NewNATS creates a new NATS that fulfill the BrokerLinker interface
func NewNATS(connection *nats.Conn) *NATS {
	return &NATS{
		connection: connection,
		queueName:  "asyncapi",
	}
}

// SetQueueName sets a custom queue name for channel subscription
//
// It can be used for multiple applications listening one the same channel but
// wants to listen on different queues.
func (c *NATS) SetQueueName(name string) {
	c.queueName = name
}

// SetLogger set a custom logger that will log operations on broker controller
func (c *NATS) SetLogger(logger log.Interface) {
	c.logger = logger
}

// Publish a message to the broker
func (c *NATS) Publish(_ context.Context, channel string, um broker.Message) error {
	msg := nats.NewMsg(channel)

	// Set message content
	msg.Data = um.Payload
	if um.CorrelationID != nil {
		msg.Header.Add(broker.CorrelationIDField.String(), *um.CorrelationID)
	}

	// Publish message
	if err := c.connection.PublishMsg(msg); err != nil {
		return err
	}

	// Flush the queue
	return c.connection.Flush()
}

// Subscribe to messages from the broker
func (c *NATS) Subscribe(ctx context.Context, channel string) (msgs chan broker.Message, stop chan interface{}, err error) {
	// Subscribe to channel
	natsMsgs := make(chan *nats.Msg, 64)
	sub, err := c.connection.QueueSubscribeSyncWithChan(channel, c.queueName, natsMsgs)
	if err != nil {
		return nil, nil, err
	}

	// Handle events
	msgs = make(chan broker.Message, 64)
	stop = make(chan interface{}, 1)
	go func() {
		for {
			select {
			// Handle new message
			case msg := <-natsMsgs:
				var correlationID *string

				// Add correlation ID if not empty
				str := msg.Header.Get(broker.CorrelationIDField.String())
				if str != "" {
					correlationID = &str
				}

				// Create message
				msgs <- broker.Message{
					Payload:       msg.Data,
					CorrelationID: correlationID,
				}
			// Handle closure request from function caller
			case <-stop:
				if err := sub.Unsubscribe(); err != nil && !errors.Is(err, nats.ErrConnectionClosed) && c.logger != nil {
					c.logger.Error(ctx, err.Error())
				}

				if err := sub.Drain(); err != nil && !errors.Is(err, nats.ErrConnectionClosed) && c.logger != nil {
					c.logger.Error(ctx, err.Error())
				}

				close(msgs)
			}
		}
	}()

	return msgs, stop, nil
}