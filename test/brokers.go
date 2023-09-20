package asyncapi_test

import (
	"testing"

	"github.com/obouchet/asyncapi-codegen/pkg/extensions"
	"github.com/obouchet/asyncapi-codegen/pkg/extensions/brokers/kafka"
	"github.com/obouchet/asyncapi-codegen/pkg/extensions/brokers/nats"
)

// BrokerControllers returns a list of BrokerController to test based on the
// docker-compose file of the project.
func BrokerControllers(t *testing.T) []extensions.BrokerController {
	t.Helper() // Set this function as a helper

	return []extensions.BrokerController{
		nats.NewController("nats://localhost:4222"),
		kafka.NewController([]string{"localhost:9094"}),
	}
}
