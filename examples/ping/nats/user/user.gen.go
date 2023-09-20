// Package "main" provides primitives to interact with the AsyncAPI specification.
//
// Code generated by github.com/obouchet/asyncapi-codegen version (devel) DO NOT EDIT.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/obouchet/asyncapi-codegen/pkg/extensions"

	"github.com/google/uuid"
)

// UserSubscriber represents all handlers that are expecting messages for User
type UserSubscriber interface {
	// Pong subscribes to messages placed on the 'pong' channel
	Pong(ctx context.Context, msg PongMessage, done bool)
}

// UserController is the structure that provides publishing capabilities to the
// developer and and connect the broker with the User
type UserController struct {
	controller
}

// NewUserController links the User to the broker
func NewUserController(bc extensions.BrokerController, options ...ControllerOption) (*UserController, error) {
	// Check if broker controller has been provided
	if bc == nil {
		return nil, ErrNilBrokerController
	}

	// Create default controller
	controller := controller{
		broker:          bc,
		stopSubscribers: make(map[string]chan interface{}),
		logger:          extensions.DummyLogger{},
		middlewares:     make([]extensions.Middleware, 0),
	}

	// Apply options
	for _, option := range options {
		option(&controller)
	}

	return &UserController{controller: controller}, nil
}

func (c UserController) wrapMiddlewares(middlewares []extensions.Middleware, last extensions.NextMiddleware) func(ctx context.Context) {
	var called bool

	// If there is no more middleware
	if len(middlewares) == 0 {
		return func(ctx context.Context) {
			if !called {
				called = true
				last(ctx)
			}
		}
	}

	// Wrap middleware into a check function that will call execute the middleware
	// and call the next wrapped middleware if the returned function has not been
	// called already
	next := c.wrapMiddlewares(middlewares[1:], last)
	return func(ctx context.Context) {
		// Call the middleware and the following if it has not been done already
		if !called {
			called = true
			ctx = middlewares[0](ctx, next)

			// If next has already been called in middleware, it should not be
			// executed again
			next(ctx)
		}
	}
}

func (c UserController) executeMiddlewares(ctx context.Context, callback func(ctx context.Context)) {
	// Wrap middleware to have 'next' function when calling them
	wrapped := c.wrapMiddlewares(c.middlewares, callback)

	// Execute wrapped middlewares
	wrapped(ctx)
}

func addUserContextValues(ctx context.Context, path string) context.Context {
	ctx = context.WithValue(ctx, extensions.ContextKeyIsProvider, "user")
	return context.WithValue(ctx, extensions.ContextKeyIsChannel, path)
}

// Close will clean up any existing resources on the controller
func (c *UserController) Close(ctx context.Context) {
	// Unsubscribing remaining channels
	c.UnsubscribeAll(ctx)
	c.logger.Info(ctx, "Closed user controller")
}

// SubscribeAll will subscribe to channels without parameters on which the app is expecting messages.
// For channels with parameters, they should be subscribed independently.
func (c *UserController) SubscribeAll(ctx context.Context, as UserSubscriber) error {
	if as == nil {
		return ErrNilUserSubscriber
	}

	if err := c.SubscribePong(ctx, as.Pong); err != nil {
		return err
	}

	return nil
}

// UnsubscribeAll will unsubscribe all remaining subscribed channels
func (c *UserController) UnsubscribeAll(ctx context.Context) {
	// Unsubscribe channels with no parameters (if any)
	c.UnsubscribePong(ctx)

	// Unsubscribe remaining channels
	for n, stopChan := range c.stopSubscribers {
		stopChan <- true
		delete(c.stopSubscribers, n)
	}
}

// SubscribePong will subscribe to new messages from 'pong' channel.
//
// Callback function 'fn' will be called each time a new message is received.
// The 'done' argument indicates when the subscription is canceled and can be
// used to clean up resources.
func (c *UserController) SubscribePong(ctx context.Context, fn func(ctx context.Context, msg PongMessage, done bool)) error {
	// Get channel path
	path := "pong"

	// Set context
	ctx = addUserContextValues(ctx, path)

	// Check if there is already a subscription
	_, exists := c.stopSubscribers[path]
	if exists {
		err := fmt.Errorf("%w: %q channel is already subscribed", ErrAlreadySubscribedChannel, path)
		c.logger.Error(ctx, err.Error())
		return err
	}

	// Subscribe to broker channel
	msgs, stop, err := c.broker.Subscribe(ctx, path)
	if err != nil {
		c.logger.Error(ctx, err.Error())
		return err
	}
	c.logger.Info(ctx, "Subscribed to channel")

	// Asynchronously listen to new messages and pass them to app subscriber
	go func() {
		for {
			// Wait for next message
			bMsg, open := <-msgs

			// Set broker message to context
			ctx = context.WithValue(ctx, extensions.ContextKeyIsBrokerMessage, bMsg)

			// Process message
			msg, err := newPongMessageFromBrokerMessage(bMsg)
			if err != nil {
				c.logger.Error(ctx, err.Error())
			}

			// Add context
			msgCtx := context.WithValue(ctx, extensions.ContextKeyIsMessage, msg)
			msgCtx = context.WithValue(msgCtx, extensions.ContextKeyIsMessageDirection, "reception")

			// Add correlation ID to context if it exists
			if id := msg.CorrelationID(); id != "" {
				ctx = context.WithValue(ctx, extensions.ContextKeyIsCorrelationID, id)
			}

			// Process message if no error and still open
			if err == nil && open {
				// Execute middlewares with the callback
				c.executeMiddlewares(msgCtx, func(ctx context.Context) {
					fn(ctx, msg, !open)
				})
			}

			// If subscription is closed, then exit the function
			if !open {
				return
			}
		}
	}()

	// Add the stop channel to the inside map
	c.stopSubscribers[path] = stop

	return nil
}

// UnsubscribePong will unsubscribe messages from 'pong' channel
func (c *UserController) UnsubscribePong(ctx context.Context) {
	// Get channel path
	path := "pong"

	// Set context
	ctx = addUserContextValues(ctx, path)

	// Get stop channel
	stopChan, exists := c.stopSubscribers[path]
	if !exists {
		return
	}

	// Stop the channel and remove the entry
	stopChan <- true
	delete(c.stopSubscribers, path)

	c.logger.Info(ctx, "Unsubscribed from channel")
}

// PublishPing will publish messages to 'ping' channel
func (c *UserController) PublishPing(ctx context.Context, msg PingMessage) error {
	// Get channel path
	path := "ping"

	// Set correlation ID if it does not exist
	if id := msg.CorrelationID(); id == "" {
		msg.SetCorrelationID(uuid.New().String())
	}

	// Set context
	ctx = addUserContextValues(ctx, path)
	ctx = context.WithValue(ctx, extensions.ContextKeyIsMessage, msg)
	ctx = context.WithValue(ctx, extensions.ContextKeyIsMessageDirection, "publication")
	ctx = context.WithValue(ctx, extensions.ContextKeyIsCorrelationID, msg.CorrelationID())

	// Convert to BrokerMessage
	bMsg, err := msg.toBrokerMessage()
	if err != nil {
		return err
	}

	// Set broker message to context
	ctx = context.WithValue(ctx, extensions.ContextKeyIsBrokerMessage, bMsg)

	// Publish the message on event-broker through middlewares
	c.executeMiddlewares(ctx, func(ctx context.Context) {
		err = c.broker.Publish(ctx, path, bMsg)
	})

	// Return error from publication on broker
	return err
}

// WaitForPong will wait for a specific message by its correlation ID
//
// The pub function is the publication function that should be used to send the message
// It will be called after subscribing to the channel to avoid race condition, and potentially loose the message
func (cc *UserController) WaitForPong(ctx context.Context, publishMsg MessageWithCorrelationID, pub func(ctx context.Context) error) (PongMessage, error) {
	// Get channel path
	path := "pong"

	// Set context
	ctx = addUserContextValues(ctx, path)

	// Subscribe to broker channel
	msgs, stop, err := cc.broker.Subscribe(ctx, path)
	if err != nil {
		cc.logger.Error(ctx, err.Error())
		return PongMessage{}, err
	}
	cc.logger.Info(ctx, "Subscribed to channel")

	// Close subscriber on leave
	defer func() {
		// Unsubscribe
		stop <- true

		// Logging unsubscribing
		cc.logger.Info(ctx, "Unsubscribed from channel")
	}()

	// Execute callback for publication
	if err = pub(ctx); err != nil {
		return PongMessage{}, err
	}

	// Wait for corresponding response
	for {
		select {
		case bMsg, open := <-msgs:
			// Get new message
			msg, err := newPongMessageFromBrokerMessage(bMsg)
			if err != nil {
				cc.logger.Error(ctx, err.Error())
			}

			// If valid message with corresponding correlation ID, return message
			if err == nil && publishMsg.CorrelationID() == msg.CorrelationID() {
				// Set context with received values
				msgCtx := context.WithValue(ctx, extensions.ContextKeyIsMessage, msg)
				msgCtx = context.WithValue(msgCtx, extensions.ContextKeyIsBrokerMessage, bMsg)
				msgCtx = context.WithValue(msgCtx, extensions.ContextKeyIsMessageDirection, "reception")
				msgCtx = context.WithValue(msgCtx, extensions.ContextKeyIsCorrelationID, publishMsg.CorrelationID())

				// Execute middlewares before returning
				cc.executeMiddlewares(msgCtx, func(_ context.Context) {
					/* Nothing to do more */
				})

				return msg, nil
			} else if !open { // If message is invalid or not corresponding and the subscription is closed, then set corresponding error
				cc.logger.Error(ctx, "Channel closed before getting message")
				return PongMessage{}, ErrSubscriptionCanceled
			}
		case <-ctx.Done(): // Set corrsponding error if context is done
			cc.logger.Error(ctx, "Context done before getting message")
			return PongMessage{}, ErrContextCanceled
		}
	}
}

// PingMessage is the message expected for 'Ping' channel
type PingMessage struct {
	// Headers will be used to fill the message headers
	Headers struct {
		// Description: Correlation ID set by user
		CorrelationID *string `json:"correlation_id"`
	}

	// Payload will be inserted in the message payload
	Payload string
}

func NewPingMessage() PingMessage {
	var msg PingMessage

	// Set correlation ID
	u := uuid.New().String()
	msg.Headers.CorrelationID = &u

	return msg
}

// newPingMessageFromBrokerMessage will fill a new PingMessage with data from generic broker message
func newPingMessageFromBrokerMessage(bMsg extensions.BrokerMessage) (PingMessage, error) {
	var msg PingMessage

	// Unmarshal payload to expected message payload format
	err := json.Unmarshal(bMsg.Payload, &msg.Payload)
	if err != nil {
		return msg, err
	}

	// Get each headers from broker message
	for k, v := range bMsg.Headers {
		switch {
		case k == "correlationId": // Retrieving CorrelationID header
			h := string(v)
			msg.Headers.CorrelationID = &h
		default:
			// TODO: log unknown error
		}
	}

	// TODO: run checks on msg type

	return msg, nil
}

// toBrokerMessage will generate a generic broker message from PingMessage data
func (msg PingMessage) toBrokerMessage() (extensions.BrokerMessage, error) {
	// TODO: implement checks on message

	// Marshal payload to JSON
	payload, err := json.Marshal(msg.Payload)
	if err != nil {
		return extensions.BrokerMessage{}, err
	}

	// Add each headers to broker message
	headers := make(map[string][]byte, 1)

	// Adding CorrelationID header
	if msg.Headers.CorrelationID != nil {
		headers["correlationId"] = []byte(*msg.Headers.CorrelationID)
	}

	return extensions.BrokerMessage{
		Headers: headers,
		Payload: payload,
	}, nil
}

// CorrelationID will give the correlation ID of the message, based on AsyncAPI spec
func (msg PingMessage) CorrelationID() string {
	if msg.Headers.CorrelationID != nil {
		return *msg.Headers.CorrelationID
	}

	return ""
}

// SetCorrelationID will set the correlation ID of the message, based on AsyncAPI spec
func (msg *PingMessage) SetCorrelationID(id string) {
	msg.Headers.CorrelationID = &id
}

// SetAsResponseFrom will correlate the message with the one passed in parameter.
// It will assign the 'req' message correlation ID to the message correlation ID,
// both specified in AsyncAPI spec.
func (msg *PingMessage) SetAsResponseFrom(req MessageWithCorrelationID) {
	id := req.CorrelationID()
	msg.Headers.CorrelationID = &id
}

// PongMessage is the message expected for 'Pong' channel
type PongMessage struct {
	// Headers will be used to fill the message headers
	Headers struct {
		// Description: Correlation ID set by user on corresponding request
		CorrelationID *string `json:"correlation_id"`
	}

	// Payload will be inserted in the message payload
	Payload struct {
		// Description: Pong message
		Message string `json:"message" validate:"required"`

		// Description: Pong creation time
		Time time.Time `json:"time" validate:"required"`
	}
}

func NewPongMessage() PongMessage {
	var msg PongMessage

	// Set correlation ID
	u := uuid.New().String()
	msg.Headers.CorrelationID = &u

	return msg
}

// newPongMessageFromBrokerMessage will fill a new PongMessage with data from generic broker message
func newPongMessageFromBrokerMessage(bMsg extensions.BrokerMessage) (PongMessage, error) {
	var msg PongMessage

	// Unmarshal payload to expected message payload format
	err := json.Unmarshal(bMsg.Payload, &msg.Payload)
	if err != nil {
		return msg, err
	}

	// Get each headers from broker message
	for k, v := range bMsg.Headers {
		switch {
		case k == "correlationId": // Retrieving CorrelationID header
			h := string(v)
			msg.Headers.CorrelationID = &h
		default:
			// TODO: log unknown error
		}
	}

	// TODO: run checks on msg type

	return msg, nil
}

// toBrokerMessage will generate a generic broker message from PongMessage data
func (msg PongMessage) toBrokerMessage() (extensions.BrokerMessage, error) {
	// TODO: implement checks on message

	// Marshal payload to JSON
	payload, err := json.Marshal(msg.Payload)
	if err != nil {
		return extensions.BrokerMessage{}, err
	}

	// Add each headers to broker message
	headers := make(map[string][]byte, 1)

	// Adding CorrelationID header
	if msg.Headers.CorrelationID != nil {
		headers["correlationId"] = []byte(*msg.Headers.CorrelationID)
	}

	return extensions.BrokerMessage{
		Headers: headers,
		Payload: payload,
	}, nil
}

// CorrelationID will give the correlation ID of the message, based on AsyncAPI spec
func (msg PongMessage) CorrelationID() string {
	if msg.Headers.CorrelationID != nil {
		return *msg.Headers.CorrelationID
	}

	return ""
}

// SetCorrelationID will set the correlation ID of the message, based on AsyncAPI spec
func (msg *PongMessage) SetCorrelationID(id string) {
	msg.Headers.CorrelationID = &id
}

// SetAsResponseFrom will correlate the message with the one passed in parameter.
// It will assign the 'req' message correlation ID to the message correlation ID,
// both specified in AsyncAPI spec.
func (msg *PongMessage) SetAsResponseFrom(req MessageWithCorrelationID) {
	id := req.CorrelationID()
	msg.Headers.CorrelationID = &id
}
