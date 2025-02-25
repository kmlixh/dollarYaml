package dollarYaml

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	ErrValueNotFound = errors.New("value not found")
	ErrLevelMismatch = errors.New("level does not match")
)

// YamlProfile represents a YAML configuration with environment variable support
type YamlProfile struct {
	data  map[string]interface{}
	debug bool
}

// Option represents a configuration option for YamlProfile
type Option func(*YamlProfile)

// WithDebug enables or disables debug logging
func WithDebug(debug bool) Option {
	return func(p *YamlProfile) {
		p.debug = debug
	}
}

// New creates a new YamlProfile instance with options
func New(opts ...Option) *YamlProfile {
	p := &YamlProfile{
		data: make(map[string]interface{}),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// SetDebug enables or disables debug logging
func (p *YamlProfile) SetDebug(debug bool) {
	p.debug = debug
}

// debugf prints debug information if debug mode is enabled
func (p *YamlProfile) debugf(format string, args ...interface{}) {
	if p.debug {
		fmt.Printf(format, args...)
	}
}

// Read unmarshals YAML data into YamlProfile
func (p *YamlProfile) Read(data []byte) error {
	var result map[string]interface{}
	if err := yaml.Unmarshal(data, &result); err != nil {
		return err
	}
	p.data = result
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

// UnmarshalTo unmarshals the YamlProfile into a target struct
// The target must be a pointer to a struct
func (p *YamlProfile) UnmarshalTo(target interface{}) error {
	if target == nil {
		return errors.New("target cannot be nil")
	}

	// Create a copy of the profile to process environment variables
	processed := make(map[string]interface{})
	if err := p.processEnvVars(p.data, processed); err != nil {
		return fmt.Errorf("processing environment variables: %w", err)
	}

	p.debugf("Processed config before marshal: %#v\n", processed)

	// Convert processed map to YAML bytes using yaml.v3 internally
	data, err := yaml.Marshal(processed)
	if err != nil {
		return fmt.Errorf("marshaling processed config: %w", err)
	}

	p.debugf("Marshaled YAML:\n%s\n", string(data))

	// Unmarshal into target struct using yaml.v3 internally
	if err := yaml.Unmarshal(data, target); err != nil {
		return fmt.Errorf("unmarshaling to target: %w", err)
	}

	return nil
}

// processEnvVars recursively processes environment variables in the configuration
func (p *YamlProfile) processEnvVars(src map[string]interface{}, dest map[string]interface{}) error {
	for k, v := range src {
		switch val := v.(type) {
		case string:
			// Process environment variables in strings
			if strings.HasPrefix(val, "${") && strings.HasSuffix(val, "}") {
				processed, _ := p.resolveValue(val)
				// Try to convert to appropriate type if the value looks like a number or boolean
				if num, err := strconv.Atoi(processed); err == nil {
					dest[k] = num
					p.debugf("Converted %s to int: %v\n", processed, num)
				} else if fnum, err := strconv.ParseFloat(processed, 64); err == nil {
					if float64(int(fnum)) == fnum {
						dest[k] = int(fnum)
						p.debugf("Converted %s to int from float: %v\n", processed, int(fnum))
					} else {
						dest[k] = fnum
						p.debugf("Converted %s to float: %v\n", processed, fnum)
					}
				} else if strings.EqualFold(processed, "true") || strings.EqualFold(processed, "false") {
					b := strings.EqualFold(processed, "true")
					dest[k] = b
					p.debugf("Converted %s to bool: %v\n", processed, b)
				} else {
					dest[k] = processed
					p.debugf("Kept as string: %s\n", processed)
				}
			} else {
				dest[k] = val
			}
		case map[string]interface{}:
			// Recursively process nested maps
			nestedDest := make(map[string]interface{})
			if err := p.processEnvVars(val, nestedDest); err != nil {
				return err
			}
			dest[k] = nestedDest
		case []interface{}:
			// Process arrays
			processed := make([]interface{}, len(val))
			for i, item := range val {
				switch itemVal := item.(type) {
				case string:
					if strings.HasPrefix(itemVal, "${") && strings.HasSuffix(itemVal, "}") {
						pval, _ := p.resolveValue(itemVal)
						// Try to convert array items as well
						if num, err := strconv.Atoi(pval); err == nil {
							processed[i] = num
							p.debugf("Array item converted %s to int: %v\n", pval, num)
						} else if fnum, err := strconv.ParseFloat(pval, 64); err == nil {
							if float64(int(fnum)) == fnum {
								processed[i] = int(fnum)
								p.debugf("Array item converted %s to int from float: %v\n", pval, int(fnum))
							} else {
								processed[i] = fnum
								p.debugf("Array item converted %s to float: %v\n", pval, fnum)
							}
						} else if strings.EqualFold(pval, "true") || strings.EqualFold(pval, "false") {
							b := strings.EqualFold(pval, "true")
							processed[i] = b
							p.debugf("Array item converted %s to bool: %v\n", pval, b)
						} else {
							processed[i] = pval
							p.debugf("Array item kept as string: %s\n", pval)
						}
					} else {
						processed[i] = itemVal
					}
				case map[string]interface{}:
					nestedDest := make(map[string]interface{})
					if err := p.processEnvVars(itemVal, nestedDest); err != nil {
						return err
					}
					processed[i] = nestedDest
				default:
					processed[i] = item
				}
			}
			dest[k] = processed
		case float64:
			// Convert float64 to int if it's a whole number
			if float64(int(val)) == val {
				dest[k] = int(val)
				p.debugf("Converted float64 %v to int: %v\n", val, int(val))
			} else {
				dest[k] = val
			}
		default:
			dest[k] = v
		}
	}
	return nil
}

// Get retrieves a value by path, returning empty string if not found
func (p *YamlProfile) Get(path string) string {
	val, _ := p.GetError(path)
	return val
}

// GetError retrieves a value by path with error handling
func (p *YamlProfile) GetError(path string) (string, error) {
	return p.get(path)
}

func (p *YamlProfile) get(path string) (string, error) {
	paths := strings.Split(path, ".")
	var current interface{} = p.data

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
func (p *YamlProfile) resolveValue(value interface{}) (string, error) {
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
