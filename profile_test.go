package dollarYaml

import (
	"errors"
	"os"
	"testing"
)

func TestYamlProfile_Read(t *testing.T) {
	yamlData := []byte(`
test:
  string: simple string
  env: ${TEST_ENV:default}
  nested:
    value: ${NESTED_VALUE:123}
    plain: plain text
`)

	var p YamlProfile
	if err := p.Read(yamlData); err != nil {
		t.Fatalf("failed to read yaml data: %v", err)
	}

	// Debug print
	t.Logf("Parsed YAML structure: %#v", p)

	tests := []struct {
		name string
		env  map[string]string
		path string
		want string

		wantErr bool
		errType error
	}{
		{
			name: "simple string value",
			path: "test.string",
			want: "simple string",
		},
		{
			name: "env variable with default",
			path: "test.env",
			want: "default",
		},
		{
			name: "env variable with value",
			env:  map[string]string{"TEST_ENV": "custom"},
			path: "test.env",
			want: "custom",
		},
		{
			name: "nested env variable with default",
			path: "test.nested.value",
			want: "123",
		},
		{
			name: "nested plain value",
			path: "test.nested.plain",
			want: "plain text",
		},
		{
			name:    "non-existent path",
			path:    "test.nonexistent",
			wantErr: true,
			errType: ErrValueNotFound,
		},
		{
			name:    "invalid path depth",
			path:    "test.string.invalid",
			wantErr: true,
			errType: ErrLevelMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.env {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			var p YamlProfile
			if err := p.Read(yamlData); err != nil {
				t.Fatalf("failed to read yaml data: %v", err)
			}

			got, err := p.GetError(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errType != nil && !errors.Is(err, tt.errType) {
					t.Errorf("expected error type %v but got %v", tt.errType, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestYamlProfile_ReadFromPath(t *testing.T) {
	// Create a temporary test file
	content := []byte(`
test:
  value: test value
  env: ${TEST_FILE_ENV:file default}
`)
	tmpfile, err := os.CreateTemp("", "test*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		env     map[string]string
		want    string
		wantErr bool
	}{
		{
			name: "read simple value from file",
			path: "test.value",
			want: "test value",
		},
		{
			name: "read env value with default from file",
			path: "test.env",
			want: "file default",
		},
		{
			name: "read env value with custom value from file",
			path: "test.env",
			env:  map[string]string{"TEST_FILE_ENV": "custom file value"},
			want: "custom file value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.env {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			var p YamlProfile
			if err := p.ReadFromPath(tmpfile.Name()); err != nil {
				t.Fatalf("failed to read from file: %v", err)
			}

			got := p.Get(tt.path)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestYamlProfile_Get_Types(t *testing.T) {
	yamlData := []byte(`
values:
  string: simple string
  number: 123
  float: 123.456
  boolean: true
  list:
    - item1
    - item2
`)

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "string value",
			path: "values.string",
			want: "simple string",
		},
		{
			name: "number value",
			path: "values.number",
			want: "123",
		},
		{
			name: "float value",
			path: "values.float",
			want: "123.456",
		},
		{
			name: "boolean value",
			path: "values.boolean",
			want: "true",
		},
	}

	var p YamlProfile
	if err := p.Read(yamlData); err != nil {
		t.Fatalf("failed to read yaml data: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.Get(tt.path)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
