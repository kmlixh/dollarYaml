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

	p := New(true) // Enable debug for this test
	if err := p.Read(yamlData); err != nil {
		t.Fatalf("failed to read yaml data: %v", err)
	}

	// Debug print
	t.Logf("Parsed YAML structure: %#v", p.data)

	tests := []struct {
		name    string
		env     map[string]string
		path    string
		want    string
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

			p := New(false) // Disable debug for individual test cases
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

			p := New(false) // Disable debug for normal operation
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

func TestYamlProfile_UnmarshalTo(t *testing.T) {
	type Database struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Options  struct {
			MaxConn int      `yaml:"maxConn"`
			Tags    []string `yaml:"tags"`
		} `yaml:"options"`
	}

	yamlData := []byte(`
database:
  host: ${DB_HOST:localhost}
  port: 5432
  user: ${DB_USER:admin}
  password: ${DB_PASSWORD:secret}
  options:
    maxConn: 100
    tags:
      - ${DB_TAG1:primary}
      - secondary
`)

	tests := []struct {
		name  string
		env   map[string]string
		want  Database
		debug bool
	}{
		{
			name:  "default values with debug",
			debug: true,
			want: Database{
				Host:     "localhost",
				Port:     5432,
				User:     "admin",
				Password: "secret",
				Options: struct {
					MaxConn int      `yaml:"maxConn"`
					Tags    []string `yaml:"tags"`
				}{
					MaxConn: 100,
					Tags:    []string{"primary", "secondary"},
				},
			},
		},
		{
			name:  "custom env values without debug",
			debug: false,
			env: map[string]string{
				"DB_HOST":     "db.example.com",
				"DB_USER":     "custom_user",
				"DB_PASSWORD": "custom_pass",
				"DB_TAG1":     "master",
			},
			want: Database{
				Host:     "db.example.com",
				Port:     5432,
				User:     "custom_user",
				Password: "custom_pass",
				Options: struct {
					MaxConn int      `yaml:"maxConn"`
					Tags    []string `yaml:"tags"`
				}{
					MaxConn: 100,
					Tags:    []string{"master", "secondary"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.env {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			p := New(tt.debug)
			if err := p.Read(yamlData); err != nil {
				t.Fatalf("failed to read yaml data: %v", err)
			}

			var config struct {
				Database Database `yaml:"database"`
			}

			if err := p.UnmarshalTo(&config); err != nil {
				t.Fatalf("UnmarshalTo failed: %v", err)
			}

			if config.Database.Host != tt.want.Host {
				t.Errorf("Host = %v, want %v", config.Database.Host, tt.want.Host)
			}
			if config.Database.Port != tt.want.Port {
				t.Errorf("Port = %v, want %v", config.Database.Port, tt.want.Port)
			}
			if config.Database.User != tt.want.User {
				t.Errorf("User = %v, want %v", config.Database.User, tt.want.User)
			}
			if config.Database.Password != tt.want.Password {
				t.Errorf("Password = %v, want %v", config.Database.Password, tt.want.Password)
			}
			if config.Database.Options.MaxConn != tt.want.Options.MaxConn {
				t.Errorf("MaxConn = %v, want %v", config.Database.Options.MaxConn, tt.want.Options.MaxConn)
			}
			if len(config.Database.Options.Tags) != len(tt.want.Options.Tags) {
				t.Errorf("Tags length = %v, want %v", len(config.Database.Options.Tags), len(tt.want.Options.Tags))
			} else {
				for i, tag := range config.Database.Options.Tags {
					if tag != tt.want.Options.Tags[i] {
						t.Errorf("Tag[%d] = %v, want %v", i, tag, tt.want.Options.Tags[i])
					}
				}
			}
		})
	}
}

func TestYamlProfile_ComplexStructures(t *testing.T) {
	type Duration struct {
		Value int    `yaml:"value"`
		Unit  string `yaml:"unit"`
	}

	type Server struct {
		Name     string            `yaml:"name"`
		Address  string            `yaml:"address"`
		Port     int               `yaml:"port"`
		Enabled  bool              `yaml:"enabled"`
		Timeout  Duration          `yaml:"timeout"`
		Tags     []string          `yaml:"tags"`
		Metadata map[string]string `yaml:"metadata"`
	}

	type Database struct {
		Master Server   `yaml:"master"`
		Slaves []Server `yaml:"slaves"`
	}

	type Config struct {
		Version    string              `yaml:"version"`
		Debug      bool                `yaml:"debug"`
		Database   Database            `yaml:"database"`
		Cache      map[string]Duration `yaml:"cache"`
		RawConfigs map[string]any      `yaml:"rawConfigs"`
		Features   []struct {
			Name        string `yaml:"name"`
			Description string `yaml:"description"`
			Enabled     bool   `yaml:"enabled"`
		} `yaml:"features"`
	}

	yamlData := []byte(`
version: ${APP_VERSION:1.0.0}
debug: ${DEBUG_MODE:true}
database:
  master:
    name: main-db
    address: ${DB_HOST:localhost}
    port: ${DB_PORT:5432}
    enabled: true
    timeout:
      value: ${DB_TIMEOUT:30}
      unit: seconds
    tags:
      - ${DB_TAG1:primary}
      - master
      - ${DB_TAG3:prod}
    metadata:
      region: ${DB_REGION:us-east}
      tier: ${DB_TIER:premium}
  slaves:
    - name: slave-1
      address: ${SLAVE1_HOST:10.0.0.1}
      port: 5432
      enabled: true
      timeout:
        value: 15
        unit: seconds
      tags:
        - replica
        - ${SLAVE1_TAG:backup}
      metadata:
        region: ${SLAVE1_REGION:us-west}
        tier: standard
cache:
  memory:
    value: ${MEMORY_CACHE_TTL:300}
    unit: seconds
  disk:
    value: ${DISK_CACHE_TTL:3600}
    unit: seconds
rawConfigs:
  limits:
    cpu: ${CPU_LIMIT:2}
    memory: ${MEMORY_LIMIT:4096}
  flags:
    feature1: true
    feature2: false
features:
  - name: auth
    description: ${AUTH_DESC:Authentication and Authorization}
    enabled: true
  - name: metrics
    description: ${METRICS_DESC:System Metrics Collection}
    enabled: true
`)

	tests := []struct {
		name     string
		env      map[string]string
		debug    bool
		validate func(*testing.T, Config)
	}{
		{
			name:  "default values with debug",
			debug: true,
			validate: func(t *testing.T, c Config) {
				// Version and debug
				assert(t, c.Version, "1.0.0", "Version")
				assert(t, c.Debug, true, "Debug")

				// Master database
				assert(t, c.Database.Master.Name, "main-db", "Master DB Name")
				assert(t, c.Database.Master.Address, "localhost", "Master DB Address")
				assert(t, c.Database.Master.Port, 5432, "Master DB Port")
				assert(t, c.Database.Master.Enabled, true, "Master DB Enabled")
				assert(t, c.Database.Master.Timeout.Value, 30, "Master DB Timeout Value")
				assert(t, c.Database.Master.Timeout.Unit, "seconds", "Master DB Timeout Unit")
				assert(t, len(c.Database.Master.Tags), 3, "Master DB Tags Length")
				assert(t, c.Database.Master.Tags[0], "primary", "Master DB Tag 1")
				assert(t, c.Database.Master.Metadata["region"], "us-east", "Master DB Region")

				// Slaves
				assert(t, len(c.Database.Slaves), 1, "Slaves Length")
				assert(t, c.Database.Slaves[0].Address, "10.0.0.1", "Slave 1 Address")
				assert(t, c.Database.Slaves[0].Enabled, true, "Slave 1 Enabled")

				// Cache
				assert(t, c.Cache["memory"].Value, 300, "Memory Cache TTL")
				assert(t, c.Cache["disk"].Value, 3600, "Disk Cache TTL")

				// Features
				assert(t, len(c.Features), 2, "Features Length")
				assert(t, c.Features[0].Name, "auth", "Feature 1 Name")
				assert(t, c.Features[0].Description, "Authentication and Authorization", "Feature 1 Description")
				assert(t, c.Features[1].Enabled, true, "Feature 2 Enabled")

				// Raw Configs
				limits, ok := c.RawConfigs["limits"].(map[string]interface{})
				if !ok {
					t.Error("Expected limits to be a map")
					return
				}
				assert(t, limits["cpu"], 2, "CPU Limit")
				assert(t, limits["memory"], 4096, "Memory Limit")

				flags, ok := c.RawConfigs["flags"].(map[string]interface{})
				if !ok {
					t.Error("Expected flags to be a map")
					return
				}
				assert(t, flags["feature1"], true, "Feature 1 Flag")
				assert(t, flags["feature2"], false, "Feature 2 Flag")
			},
		},
		{
			name:  "custom env values without debug",
			debug: false,
			env: map[string]string{
				"APP_VERSION":      "2.0.0",
				"DEBUG_MODE":       "false",
				"DB_HOST":          "custom-db.example.com",
				"DB_PORT":          "6543",
				"DB_TIMEOUT":       "45",
				"DB_TAG1":          "custom-primary",
				"DB_REGION":        "eu-central",
				"SLAVE1_HOST":      "slave1.example.com",
				"MEMORY_CACHE_TTL": "600",
				"CPU_LIMIT":        "4",
				"METRICS_DESC":     "Custom Metrics System",
			},
			validate: func(t *testing.T, c Config) {
				// Version and debug
				assert(t, c.Version, "2.0.0", "Version")
				assert(t, c.Debug, false, "Debug")

				// Master database
				assert(t, c.Database.Master.Address, "custom-db.example.com", "Master DB Address")
				assert(t, c.Database.Master.Port, 6543, "Master DB Port")
				assert(t, c.Database.Master.Timeout.Value, 45, "Master DB Timeout Value")
				assert(t, c.Database.Master.Tags[0], "custom-primary", "Master DB Tag 1")
				assert(t, c.Database.Master.Metadata["region"], "eu-central", "Master DB Region")

				// Slaves
				assert(t, c.Database.Slaves[0].Address, "slave1.example.com", "Slave 1 Address")

				// Cache
				assert(t, c.Cache["memory"].Value, 600, "Memory Cache TTL")

				// Raw configs
				limits, ok := c.RawConfigs["limits"].(map[string]interface{})
				if !ok {
					t.Error("Expected limits to be a map")
					return
				}
				assert(t, limits["cpu"], 4, "CPU Limit")

				// Features
				assert(t, c.Features[1].Description, "Custom Metrics System", "Metrics Description")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.env {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			p := New(tt.debug)
			if err := p.Read(yamlData); err != nil {
				t.Fatalf("failed to read yaml data: %v", err)
			}

			var config Config
			if err := p.UnmarshalTo(&config); err != nil {
				t.Fatalf("UnmarshalTo failed: %v", err)
			}

			tt.validate(t, config)
		})
	}
}

// assert is a test helper function to reduce test boilerplate
func assert(t *testing.T, got, want interface{}, msg string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", msg, got, want)
	}
}
