package codegen

import (
	"encoding/json"
	"fmt"
)

const (
	goTag          = "x-go-tag"
	goTagValidator = "x-go-tag-validator"
)

func extString(extPropValue interface{}) (string, error) {
	raw, ok := extPropValue.(json.RawMessage)
	if !ok {
		return "", fmt.Errorf("failed to convert type: %T", extPropValue)
	}
	var str string
	if err := json.Unmarshal(raw, &str); err != nil {
		return "", fmt.Errorf("failed to unmarshal json: %w", err)
	}

	return str, nil
}
