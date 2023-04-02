// Package "oneof" provides primitives to interact with the AsyncAPI specification.
//
// Code generated by github.com/lerenn/asyncapi-codegen version (devel) DO NOT EDIT.
package oneof

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AppSubscriber represents all handlers that are expecting messages for App
type AppSubscriber interface {
	// Test
	Test(msg TestMessagesMessage, done bool)
}

// AppController is the structure that provides publishing capabilities to the
// developer and and connect the broker with the App
type AppController struct {
	brokerController BrokerController
	stopSubscribers  map[string]chan interface{}
	logger           Logger
}

// NewAppController links the App to the broker
func NewAppController(bs BrokerController) (*AppController, error) {
	if bs == nil {
		return nil, ErrNilBrokerController
	}

	return &AppController{
		brokerController: bs,
		stopSubscribers:  make(map[string]chan interface{}),
	}, nil
}

// AttachLogger attaches a logger that will log operations on controller
func (c *AppController) AttachLogger(logger Logger) {
	c.logger = logger
	c.brokerController.AttachLogger(logger)
}

// logError logs error if the logger has been set
func (c AppController) logError(msg string, keyvals ...interface{}) {
	if c.logger != nil {
		keyvals = append(keyvals, "module", "asyncapi", "controller", "App")
		c.logger.Error(msg, keyvals...)
	}
}

// logInfo logs information if the logger has been set
func (c AppController) logInfo(msg string, keyvals ...interface{}) {
	if c.logger != nil {
		keyvals = append(keyvals, "module", "asyncapi", "controller", "App")
		c.logger.Info(msg, keyvals...)
	}
}

// Close will clean up any existing resources on the controller
func (c *AppController) Close() {
	// Unsubscribing remaining channels
	c.logInfo("Closing App controller")
	c.UnsubscribeAll()
}

// SubscribeAll will subscribe to channels without parameters on which the app is expecting messages.
// For channels with parameters, they should be subscribed independently.
func (c *AppController) SubscribeAll(as AppSubscriber) error {
	if as == nil {
		return ErrNilAppSubscriber
	}

	if err := c.SubscribeTest(as.Test); err != nil {
		return err
	}

	return nil
}

// UnsubscribeAll will unsubscribe all remaining subscribed channels
func (c *AppController) UnsubscribeAll() {
	// Unsubscribe channels with no parameters (if any)
	c.UnsubscribeTest()

	// Unsubscribe remaining channels
	for n, stopChan := range c.stopSubscribers {
		stopChan <- true
		delete(c.stopSubscribers, n)
	}
}

// SubscribeTest will subscribe to new messages from 'test' channel.
//
// Callback function 'fn' will be called each time a new message is received.
// The 'done' argument indicates when the subscription is canceled and can be
// used to clean up resources.
func (c *AppController) SubscribeTest(fn func(msg TestMessagesMessage, done bool)) error {
	// Get channel path
	path := "test"

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
			msg, err := newTestMessagesMessageFromUniversalMessage(um)
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

// UnsubscribeTest will unsubscribe messages from 'test' channel
func (c *AppController) UnsubscribeTest() {
	// Get channel path
	path := "test"

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

// PublishTest2 will publish messages to 'test2' channel
func (c *AppController) PublishTest2(msg Test2Message) error {
	// Convert to UniversalMessage
	um, err := msg.toUniversalMessage()
	if err != nil {
		return err
	}

	// Get channel path
	path := "test2"

	// Publish on event broker
	c.logInfo("Publishing to channel", "channel", path, "operation", "publish", "message", msg)
	return c.brokerController.Publish(path, um)
}

// ClientSubscriber represents all handlers that are expecting messages for Client
type ClientSubscriber interface {
	// Test2
	Test2(msg Test2Message, done bool)
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

	if err := c.SubscribeTest2(as.Test2); err != nil {
		return err
	}

	return nil
}

// UnsubscribeAll will unsubscribe all remaining subscribed channels
func (c *ClientController) UnsubscribeAll() {
	// Unsubscribe channels with no parameters (if any)
	c.UnsubscribeTest2()

	// Unsubscribe remaining channels
	for n, stopChan := range c.stopSubscribers {
		stopChan <- true
		delete(c.stopSubscribers, n)
	}
}

// SubscribeTest2 will subscribe to new messages from 'test2' channel.
//
// Callback function 'fn' will be called each time a new message is received.
// The 'done' argument indicates when the subscription is canceled and can be
// used to clean up resources.
func (c *ClientController) SubscribeTest2(fn func(msg Test2Message, done bool)) error {
	// Get channel path
	path := "test2"

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
			msg, err := newTest2MessageFromUniversalMessage(um)
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

// UnsubscribeTest2 will unsubscribe messages from 'test2' channel
func (c *ClientController) UnsubscribeTest2() {
	// Get channel path
	path := "test2"

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

// PublishTest will publish messages to 'test' channel
func (c *ClientController) PublishTest(msg TestMessagesMessage) error {
	// Convert to UniversalMessage
	um, err := msg.toUniversalMessage()
	if err != nil {
		return err
	}

	// Get channel path
	path := "test"

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

// Test2Message is the message expected for 'Test2' channel
type Test2Message struct {
	// Payload will be inserted in the message payload
	Payload struct {
		Key2 *string `json:"key2"`
	}
}

func NewTest2Message() Test2Message {
	var msg Test2Message

	return msg
}

// newTest2MessageFromUniversalMessage will fill a new Test2Message with data from UniversalMessage
func newTest2MessageFromUniversalMessage(um UniversalMessage) (Test2Message, error) {
	var msg Test2Message

	// Unmarshal payload to expected message payload format
	err := json.Unmarshal(um.Payload, &msg.Payload)
	if err != nil {
		return msg, err
	}

	// TODO: run checks on msg type

	return msg, nil
}

// toUniversalMessage will generate an UniversalMessage from Test2Message data
func (msg Test2Message) toUniversalMessage() (UniversalMessage, error) {
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

// TestMessage1Message is the message expected for 'TestMessage1' channel
type TestMessage1Message struct {
	// Payload will be inserted in the message payload
	Payload ObjectWithKeySchema
}

func NewTestMessage1Message() TestMessage1Message {
	var msg TestMessage1Message

	return msg
}

// newTestMessage1MessageFromUniversalMessage will fill a new TestMessage1Message with data from UniversalMessage
func newTestMessage1MessageFromUniversalMessage(um UniversalMessage) (TestMessage1Message, error) {
	var msg TestMessage1Message

	// Unmarshal payload to expected message payload format
	err := json.Unmarshal(um.Payload, &msg.Payload)
	if err != nil {
		return msg, err
	}

	// TODO: run checks on msg type

	return msg, nil
}

// toUniversalMessage will generate an UniversalMessage from TestMessage1Message data
func (msg TestMessage1Message) toUniversalMessage() (UniversalMessage, error) {
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

// TestMessage2Message is the message expected for 'TestMessage2' channel
type TestMessage2Message struct {
	// Payload will be inserted in the message payload
	Payload ObjectWithKey2Schema
}

func NewTestMessage2Message() TestMessage2Message {
	var msg TestMessage2Message

	return msg
}

// newTestMessage2MessageFromUniversalMessage will fill a new TestMessage2Message with data from UniversalMessage
func newTestMessage2MessageFromUniversalMessage(um UniversalMessage) (TestMessage2Message, error) {
	var msg TestMessage2Message

	// Unmarshal payload to expected message payload format
	err := json.Unmarshal(um.Payload, &msg.Payload)
	if err != nil {
		return msg, err
	}

	// TODO: run checks on msg type

	return msg, nil
}

// toUniversalMessage will generate an UniversalMessage from TestMessage2Message data
func (msg TestMessage2Message) toUniversalMessage() (UniversalMessage, error) {
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

// TestMessagesMessage is the message expected for 'TestMessages' channel
type TestMessagesMessage struct {
	// Payload will be inserted in the message payload
	Payload struct {
		Key  *string `json:"key"`
		Key2 *string `json:"key2"`
	}
}

func NewTestMessagesMessage() TestMessagesMessage {
	var msg TestMessagesMessage

	return msg
}

// newTestMessagesMessageFromUniversalMessage will fill a new TestMessagesMessage with data from UniversalMessage
func newTestMessagesMessageFromUniversalMessage(um UniversalMessage) (TestMessagesMessage, error) {
	var msg TestMessagesMessage

	// Unmarshal payload to expected message payload format
	err := json.Unmarshal(um.Payload, &msg.Payload)
	if err != nil {
		return msg, err
	}

	// TODO: run checks on msg type

	return msg, nil
}

// toUniversalMessage will generate an UniversalMessage from TestMessagesMessage data
func (msg TestMessagesMessage) toUniversalMessage() (UniversalMessage, error) {
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

// ObjectWithKeySchema is a schema from the AsyncAPI specification required in messages
type ObjectWithKeySchema struct {
	Key *string `json:"key"`
}

// ObjectWithKey2Schema is a schema from the AsyncAPI specification required in messages
type ObjectWithKey2Schema struct {
	Key2 *string `json:"key2"`
}
