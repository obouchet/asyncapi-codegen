package asyncapi

import (
	"strconv"
	"strings"

	"github.com/mohae/deepcopy"
	"github.com/obouchet/asyncapi-codegen/pkg/utils"
)

// Message is a representation of the corresponding asyncapi object filled
// from an asyncapi specification that will be used to generate code.
// Source: https://www.asyncapi.com/docs/reference/specification/v2.6.0#messageObject
type Message struct {
	Description   string         `json:"description"`
	Headers       *Schema        `json:"headers"`
	OneOf         []*Message     `json:"oneOf"`
	Payload       *Schema        `json:"payload"`
	CorrelationID *CorrelationID `json:"correlationID"`
	Reference     string         `json:"$ref"`

	// --- Non AsyncAPI fields -------------------------------------------------
	Name        string   `json:"-"`
	ReferenceTo *Message `json:"-"`

	// CorrelationIDLocation will indicate where the correlation id is
	// According to: https://www.asyncapi.com/docs/reference/specification/v2.6.0#correlationIDObject
	CorrelationIDLocation string `json:"-"`
	CorrelationIDRequired bool   `json:"-"`
}

// Process processes the Message to make it ready for code generation.
func (msg *Message) Process(name string, spec Specification, useStandardGoJson bool) {
	msg.Name = utils.UpperFirstLetter(name)

	// Add pointer to reference if there is one
	if msg.Reference != "" {
		msg.ReferenceTo = spec.ReferenceMessage(msg.Reference)
	}

	// Process Headers and Payload
	if msg.Headers != nil {
		msg.Headers.Process(name+"Headers", spec, false, useStandardGoJson)
	}
	if msg.Payload != nil {
		msg.Payload.Process(name+"Payload", spec, false, useStandardGoJson)
	}

	// Process OneOf
	for k, v := range msg.OneOf {
		// Process the OneOf
		v.Process(name+strconv.Itoa(k), spec, useStandardGoJson)

		// Merge the OneOf as one payload
		msg.MergeWith(spec, *v)
	}

	// Process correlation ID
	msg.createCorrelationIDFieldIfMissing()
	msg.CorrelationIDLocation = msg.getCorrelationIDLocation(spec)
	msg.CorrelationIDRequired = msg.isCorrelationIDRequired()
}

func (msg Message) getCorrelationIDLocation(spec Specification) string {
	// Let's check the message before the reference
	if msg.CorrelationID != nil {
		return msg.CorrelationID.Location
	}

	// If there is a reference, check it
	if msg.Reference != "" {
		correlationID := spec.ReferenceMessage(msg.Reference).CorrelationID
		if correlationID != nil {
			return correlationID.Location
		}
	}

	return ""
}

func (msg Message) isCorrelationIDRequired() bool {
	if msg.CorrelationID == nil || msg.CorrelationID.Location == "" {
		return false
	}

	correlationIDParent := msg.createTreeUntilCorrelationID()
	path := strings.Split(msg.CorrelationID.Location, "/")
	return correlationIDParent.IsFieldRequired(path[len(path)-1])
}

func (msg *Message) createCorrelationIDFieldIfMissing() {
	_ = msg.createTreeUntilCorrelationID()
}

func (msg *Message) createTreeUntilCorrelationID() (correlationIDParent *Schema) {
	// Check that correlationID exists
	if msg.CorrelationID == nil || msg.CorrelationID.Location == "" {
		return utils.ToPointer(NewSchema())
	}

	// Get root from header or payload
	var child *Schema
	path := strings.Split(msg.CorrelationID.Location, "/")
	if strings.HasPrefix(msg.CorrelationID.Location, "$message.header#") {
		if msg.Headers == nil {
			msg.Headers = utils.ToPointer(NewSchema())
			msg.Headers.Name = TypeIsHeader.String()
			msg.Headers.Type = TypeIsObject.String()
		}
		child = msg.Headers
	} else if strings.HasPrefix(msg.CorrelationID.Location, "$message.payload#") && msg.Payload != nil {
		if msg.Payload == nil {
			msg.Payload = utils.ToPointer(NewSchema())
			msg.Payload.Name = TypeIsHeader.String()
			msg.Payload.Type = TypeIsObject.String()
		}
		child = msg.Payload
	}

	// Go down the path to correlation ID
	return downToCorrelationID(path, child)
}

func downToCorrelationID(path []string, child *Schema) (correlationIDParent *Schema) {
	var exists bool
	for i, v := range path[1:] {
		correlationIDParent = child
		child, exists = correlationIDParent.Properties[v]
		if !exists {
			// Create child
			child = utils.ToPointer(NewSchema())
			child.Name = v
			if i == len(path)-2 { // As there is -1 in the loop slice
				child.Type = TypeIsString.String()
			} else {
				child.Type = TypeIsHeader.String()
			}

			// Add it to parent
			if correlationIDParent.Properties == nil {
				correlationIDParent.Properties = make(map[string]*Schema)
			}
			correlationIDParent.Properties[v] = child
		}
	}

	return correlationIDParent
}

func (msg *Message) referenceFrom(ref []string) interface{} {
	if len(ref) == 0 {
		return msg
	}

	var next *Schema
	if ref[0] == "payload" {
		next = msg.Payload
	} else if ref[0] == TypeIsHeader.String() {
		next = msg.Headers
	}

	return next.referenceFrom(ref[1:])
}

// MergeWith merges the Message with another one.
func (msg *Message) MergeWith(spec Specification, msg2 Message) {
	// Remove reference if merging
	if msg.Reference != "" {
		refMsg := spec.ReferenceMessage(msg.Reference)
		msg.Reference = ""
		msg.MergeWith(spec, *refMsg)
	}

	// Get reference from msg2
	if msg2.Reference != "" {
		refMsg2 := spec.ReferenceMessage(msg2.Reference)
		msg2.MergeWith(spec, *refMsg2)
	}

	// Merge Payload
	if msg2.Payload != nil {
		if msg.Payload == nil {
			msg.Payload = deepcopy.Copy(msg2.Payload).(*Schema)
		} else {
			msg.Payload.MergeWith(spec, *msg2.Payload)
		}
	}

	// Merge Headers
	if msg2.Headers != nil {
		if msg.Headers == nil {
			msg.Headers = deepcopy.Copy(msg2.Headers).(*Schema)
		} else {
			msg.Headers.MergeWith(spec, *msg2.Headers)
		}
	}
}
