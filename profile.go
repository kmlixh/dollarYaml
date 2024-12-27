package dollarYaml

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	ErrValueNotFound = errors.New("value not found")
	ErrLevelMismatch = errors.New("level does not match")
)

// YamlProfile represents a YAML configuration with environment variable support
type YamlProfile map[string]interface{}

// Read unmarshals YAML data into YamlProfile
func (p *YamlProfile) Read(data []byte) error {
	var result map[string]interface{}
	if err := yaml.Unmarshal(data, &result); err != nil {
		return err
	}
	*p = result
	return nil
}

// ReadFromPath reads and unmarshals YAML from a file path
func (p *YamlProfile) ReadFromPath(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}
	return p.Read(data)
}

// Get retrieves a value by path, returning empty string if not found
func (p YamlProfile) Get(path string) string {
	val, _ := p.GetError(path)
	return val
}

// GetError retrieves a value by path with error handling
func (p YamlProfile) GetError(path string) (string, error) {
	return p.get(path)
}

func (p YamlProfile) get(path string) (string, error) {
	paths := strings.Split(path, ".")
	var current interface{} = map[string]interface{}(p)

	for i, key := range paths {
		currentMap, ok := current.(map[string]interface{})
		if !ok {
			return "", ErrLevelMismatch
		}

		value, ok := currentMap[key]
		if !ok {
			return "", fmt.Errorf("%w: %s", ErrValueNotFound, key)
		}

		isLastElement := i == len(paths)-1
		if isLastElement {
			return p.resolveValue(value)
		}

		current = value
	}

	return "", ErrValueNotFound
}

// resolveValue handles the conversion and environment variable resolution
func (p YamlProfile) resolveValue(value interface{}) (string, error) {
	// Handle non-string values
	if str, ok := value.(string); ok {
		if !strings.HasPrefix(str, "${") || !strings.HasSuffix(str, "}") {
			return str, nil
		}

		// Strip ${} markers
		envStr := str[2 : len(str)-1]
		if colonIdx := strings.Index(envStr, ":"); colonIdx != -1 {
			envName := envStr[:colonIdx]
			if envValue := os.Getenv(envName); envValue != "" {
				return envValue, nil
			}
			return envStr[colonIdx+1:], nil
		}

		return os.Getenv(envStr), nil
	}

	return fmt.Sprint(value), nil
}
