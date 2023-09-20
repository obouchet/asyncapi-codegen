package codegen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/obouchet/asyncapi-codegen/pkg/asyncapi"
)

// FromFile parses the AsyncAPI specification either from a YAML file or a JSON file.
func FromFile(path string, useStandardGoJson bool) (CodeGen, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return CodeGen{}, err
	}

	switch filepath.Ext(path) {
	case ".yaml", ".yml":
		return FromYAML(data, useStandardGoJson)
	case ".json":
		return FromJSON(data, useStandardGoJson)
	default:
		return CodeGen{}, fmt.Errorf("%w: %q", ErrInvalidFileFormat, path)
	}
}

// FromYAML parses the AsyncAPI specification from a YAML file.
func FromYAML(data []byte, useStandardGoJson bool) (CodeGen, error) {
	data, err := yaml.YAMLToJSON(data)
	if err != nil {
		return CodeGen{}, err
	}

	return FromJSON(data, useStandardGoJson)
}

// FromJSON parses the AsyncAPI specification from a JSON file.
func FromJSON(data []byte, useStandardGoJson bool) (CodeGen, error) {
	var spec asyncapi.Specification

	if err := json.Unmarshal(data, &spec); err != nil {
		return CodeGen{}, err
	}

	spec.Process(useStandardGoJson)

	return New(spec), nil
}
