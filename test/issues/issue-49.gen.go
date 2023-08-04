// Package "issues" provides primitives to interact with the AsyncAPI specification.
//
// Code generated by github.com/lerenn/asyncapi-codegen version (devel) DO NOT EDIT.
package issues

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ClientSubscriber represents all handlers that are expecting messages for Client
type ClientSubscriber interface {
	// Chat
	Chat(msg ChatMessage, done bool)

	// Status
	Status(msg StatusMessage, done bool)
}

// ClientController is the structure that provides publishing capabilities to the
// developer and and connect the broker with the Client
type ClientController struct {
	brokerController BrokerController
	stopSubscribers  map[string]chan interface{}
	logger           Logger
}

// NewClientController links the Client to the broker
func NewClientController(bs BrokerController) (*ClientController, error) {
	if bs == nil {
		return nil, ErrNilBrokerController
	}

	return &ClientController{
		brokerController: bs,
		stopSubscribers:  make(map[string]chan interface{}),
	}, nil
}

// AttachLogger attaches a logger that will log operations on controller
func (c *ClientController) AttachLogger(logger Logger) {
	c.logger = logger
	c.brokerController.AttachLogger(logger)
}

// logError logs error if the logger has been set
func (c ClientController) logError(msg string, keyvals ...interface{}) {
	if c.logger != nil {
		keyvals = append(keyvals, "module", "asyncapi", "controller", "Client")
		c.logger.Error(msg, keyvals...)
	}
}

// logInfo logs information if the logger has been set
func (c ClientController) logInfo(msg string, keyvals ...interface{}) {
	if c.logger != nil {
		keyvals = append(keyvals, "module", "asyncapi", "controller", "Client")
		c.logger.Info(msg, keyvals...)
	}
}

// Close will clean up any existing resources on the controller
func (c *ClientController) Close() {
	// Unsubscribing remaining channels
	c.logInfo("Closing Client controller")
	c.UnsubscribeAll()
}

// SubscribeAll will subscribe to channels without parameters on which the app is expecting messages.
// For channels with parameters, they should be subscribed independently.
func (c *ClientController) SubscribeAll(as ClientSubscriber) error {
	if as == nil {
		return ErrNilClientSubscriber
	}

	if err := c.SubscribeStatus(as.Status); err != nil {
		return err
	}

	return nil
}

// UnsubscribeAll will unsubscribe all remaining subscribed channels
func (c *ClientController) UnsubscribeAll() {
	// Unsubscribe channels with no parameters (if any)
	c.UnsubscribeStatus()

	// Unsubscribe remaining channels
	for n, stopChan := range c.stopSubscribers {
		stopChan <- true
		delete(c.stopSubscribers, n)
	}
}

// SubscribeStatus will subscribe to new messages from '/status' channel.
//
// Callback function 'fn' will be called each time a new message is received.
// The 'done' argument indicates when the subscription is canceled and can be
// used to clean up resources.
func (c *ClientController) SubscribeStatus(fn func(msg StatusMessage, done bool)) error {
	// Get channel path
	path := "/status"

	// Check if there is already a subscription
	_, exists := c.stopSubscribers[path]
	if exists {
		err := fmt.Errorf("%w: %q channel is already subscribed", ErrAlreadySubscribedChannel, path)
		c.logError(err.Error(), "channel", path)
		return err
	}

	// Subscribe to broker channel
	c.logInfo("Subscribing to channel", "channel", path, "operation", "subscribe")
	msgs, stop, err := c.brokerController.Subscribe(path)
	if err != nil {
		c.logError(err.Error(), "channel", path, "operation", "subscribe")
		return err
	}

	// Asynchronously listen to new messages and pass them to app subscriber
	go func() {
		for {
			// Wait for next message
			um, open := <-msgs

			// Process message
			msg, err := newStatusMessageFromUniversalMessage(um)
			if err != nil {
				c.logError(err.Error(), "channel", path, "operation", "subscribe", "message", msg)
			}

			// Send info if message is correct or susbcription is closed
			if err == nil || !open {
				c.logInfo("Received new message", "channel", path, "operation", "subscribe", "message", msg)
				fn(msg, !open)
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

// UnsubscribeStatus will unsubscribe messages from '/status' channel
func (c *ClientController) UnsubscribeStatus() {
	// Get channel path
	path := "/status"

	// Get stop channel
	stopChan, exists := c.stopSubscribers[path]
	if !exists {
		return
	}

	// Stop the channel and remove the entry
	c.logInfo("Unsubscribing from channel", "channel", path, "operation", "unsubscribe")
	stopChan <- true
	delete(c.stopSubscribers, path)
}

// PublishChat will publish messages to '/chat' channel
func (c *ClientController) PublishChat(msg ChatMessage) error {
	// Convert to UniversalMessage
	um, err := msg.toUniversalMessage()
	if err != nil {
		return err
	}

	// Get channel path
	path := "/chat"

	// Publish on event broker
	c.logInfo("Publishing to channel", "channel", path, "operation", "publish", "message", msg)
	return c.brokerController.Publish(path, um)
}

const (
	// CorrelationIDField is the name of the field that will contain the correlation ID
	CorrelationIDField = "correlation_id"
)

// UniversalMessage is a wrapper that will contain all information regarding a message
type UniversalMessage struct {
	CorrelationID *string
	Payload       []byte
}

// BrokerController represents the functions that should be implemented to connect
// the broker to the application or the client
type BrokerController interface {
	// AttachLogger attaches a logger that will log operations on broker controller
	AttachLogger(logger Logger)

	// Publish a message to the broker
	Publish(channel string, mw UniversalMessage) error

	// Subscribe to messages from the broker
	Subscribe(channel string) (msgs chan UniversalMessage, stop chan interface{}, err error)
}

var (
	// Generic error for AsyncAPI generated code
	ErrAsyncAPI = errors.New("error when using AsyncAPI")

	// ErrContextCanceled is given when a given context is canceled
	ErrContextCanceled = fmt.Errorf("%w: context canceled", ErrAsyncAPI)

	// ErrNilBrokerController is raised when a nil broker controller is user
	ErrNilBrokerController = fmt.Errorf("%w: nil broker controller has been used", ErrAsyncAPI)

	// ErrNilAppSubscriber is raised when a nil app subscriber is user
	ErrNilAppSubscriber = fmt.Errorf("%w: nil app subscriber has been used", ErrAsyncAPI)

	// ErrNilClientSubscriber is raised when a nil client subscriber is user
	ErrNilClientSubscriber = fmt.Errorf("%w: nil client subscriber has been used", ErrAsyncAPI)

	// ErrAlreadySubscribedChannel is raised when a subscription is done twice
	// or more without unsubscribing
	ErrAlreadySubscribedChannel = fmt.Errorf("%w: the channel has already been subscribed", ErrAsyncAPI)

	// ErrSubscriptionCanceled is raised when expecting something and the subscription has been canceled before it happens
	ErrSubscriptionCanceled = fmt.Errorf("%w: the subscription has been canceled", ErrAsyncAPI)
)

type Logger interface {
	// Info logs information based on a message and key-value elements
	Info(msg string, keyvals ...interface{})

	// Error logs error based on a message and key-value elements
	Error(msg string, keyvals ...interface{})
}

type MessageWithCorrelationID interface {
	CorrelationID() string
}

type Error struct {
	Channel string
	Err     error
}

func (e *Error) Error() string {
	return fmt.Sprintf("channel %q: err %v", e.Channel, e.Err)
}

// ChatMessage is the message expected for 'Chat' channel
type ChatMessage struct {
	// Payload will be inserted in the message payload
	Payload string
}

func NewChatMessage() ChatMessage {
	var msg ChatMessage

	return msg
}

// newChatMessageFromUniversalMessage will fill a new ChatMessage with data from UniversalMessage
func newChatMessageFromUniversalMessage(um UniversalMessage) (ChatMessage, error) {
	var msg ChatMessage

	// Unmarshal payload to expected message payload format
	err := json.Unmarshal(um.Payload, &msg.Payload)
	if err != nil {
		return msg, err
	}

	// TODO: run checks on msg type

	return msg, nil
}

// toUniversalMessage will generate an UniversalMessage from ChatMessage data
func (msg ChatMessage) toUniversalMessage() (UniversalMessage, error) {
	// TODO: implement checks on message

	// Marshal payload to JSON
	payload, err := json.Marshal(msg.Payload)
	if err != nil {
		return UniversalMessage{}, err
	}

	return UniversalMessage{
		Payload: payload,
	}, nil
}

// Message is the message expected for ” channel
type Message struct {
	// Payload will be inserted in the message payload
	Payload string
}

func NewMessage() Message {
	var msg Message

	return msg
}

// newMessageFromUniversalMessage will fill a new Message with data from UniversalMessage
func newMessageFromUniversalMessage(um UniversalMessage) (Message, error) {
	var msg Message

	// Unmarshal payload to expected message payload format
	err := json.Unmarshal(um.Payload, &msg.Payload)
	if err != nil {
		return msg, err
	}

	// TODO: run checks on msg type

	return msg, nil
}

// toUniversalMessage will generate an UniversalMessage from Message data
func (msg Message) toUniversalMessage() (UniversalMessage, error) {
	// TODO: implement checks on message

	// Marshal payload to JSON
	payload, err := json.Marshal(msg.Payload)
	if err != nil {
		return UniversalMessage{}, err
	}

	return UniversalMessage{
		Payload: payload,
	}, nil
}

// StatusMessage is the message expected for 'Status' channel
type StatusMessage struct {
	// Payload will be inserted in the message payload
	Payload string
}

func NewStatusMessage() StatusMessage {
	var msg StatusMessage

	return msg
}

// newStatusMessageFromUniversalMessage will fill a new StatusMessage with data from UniversalMessage
func newStatusMessageFromUniversalMessage(um UniversalMessage) (StatusMessage, error) {
	var msg StatusMessage

	// Unmarshal payload to expected message payload format
	err := json.Unmarshal(um.Payload, &msg.Payload)
	if err != nil {
		return msg, err
	}

	// TODO: run checks on msg type

	return msg, nil
}

// toUniversalMessage will generate an UniversalMessage from StatusMessage data
func (msg StatusMessage) toUniversalMessage() (UniversalMessage, error) {
	// TODO: implement checks on message

	// Marshal payload to JSON
	payload, err := json.Marshal(msg.Payload)
	if err != nil {
		return UniversalMessage{}, err
	}

	return UniversalMessage{
		Payload: payload,
	}, nil
}