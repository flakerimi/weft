package warp

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Config is what an app is configured with: the listening address, the body
// limit, and any other key the app reads with Get or On. The zero Config
// works: every interface, port 8080, 1 MiB.
type Config struct {
	Host         string // HOST; empty is every interface
	Port         string // PORT; default "8080"
	MaxBodyBytes int64  // MAX_BODY_MB x 1 MiB; default 1 MiB

	vars map[string]string
	err  error
}

const (
	defaultPort         = "8080"
	defaultMaxBodyBytes = int64(1 << 20)
)

var envKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// FromEnv reads ./.env if it exists, then the environment, which wins.
func FromEnv() Config {
	vars := map[string]string{}
	var problems []error
	if data, err := os.ReadFile(".env"); err == nil {
		problems = parseDotEnv(string(data), vars)
	}
	for _, kv := range os.Environ() {
		if k, v, ok := strings.Cut(kv, "="); ok {
			vars[k] = v
		}
	}
	return build(vars, problems)
}

// ConfigFrom is FromEnv's result from a map, for tests and for apps that
// configure themselves.
func ConfigFrom(vars map[string]string) Config {
	copied := make(map[string]string, len(vars))
	for k, v := range vars {
		copied[k] = v
	}
	return build(copied, nil)
}

// Get returns the value of key, or "" when it is unset.
func (c Config) Get(key string) string { return c.vars[key] }

// On reports whether key is on, true, 1 or yes, in any case.
func (c Config) On(key string) bool {
	switch strings.ToLower(strings.TrimSpace(c.vars[key])) {
	case "on", "true", "1", "yes":
		return true
	}
	return false
}

// Err reports every problem found while reading the configuration, or nil.
func (c Config) Err() error { return c.err }

// withDefaults returns c with the port and the body limit filled in.
func (c Config) withDefaults() Config {
	if c.Port == "" {
		c.Port = defaultPort
	}
	if c.MaxBodyBytes == 0 {
		c.MaxBodyBytes = defaultMaxBodyBytes
	}
	return c
}

func build(vars map[string]string, problems []error) Config {
	c := Config{Host: vars["HOST"], vars: vars}
	if p := vars["PORT"]; p != "" {
		if n, err := strconv.Atoi(p); err == nil && n >= 1 && n <= 65535 {
			c.Port = p
		} else {
			problems = append(problems, fmt.Errorf("PORT is %q; set it to a number from 1 to 65535 in .env or the environment.", p))
		}
	}
	if m := vars["MAX_BODY_MB"]; m != "" {
		if n, err := strconv.ParseInt(m, 10, 64); err == nil && n >= 1 && n <= (1<<62)>>20 {
			c.MaxBodyBytes = n << 20
		} else {
			problems = append(problems, fmt.Errorf("MAX_BODY_MB is %q; set it to a whole number of megabytes, 1 or more, in .env or the environment.", m))
		}
	}
	c.err = errors.Join(problems...)
	return c
}

// parseDotEnv reads the lines of a .env file into vars and returns one error
// for each line it cannot read.
func parseDotEnv(text string, vars map[string]string) []error {
	var problems []error
	for i, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		key, value, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || !envKey.MatchString(key) {
			problems = append(problems, fmt.Errorf(".env line %d is %q; write it as KEY=value, or start it with # to make it a comment.", i+1, line))
			continue
		}
		vars[key] = parseValue(value)
	}
	return problems
}

func parseValue(v string) string {
	v = strings.TrimSpace(v)
	if v != "" && (v[0] == '"' || v[0] == '\'') {
		if end := strings.IndexByte(v[1:], v[0]); end >= 0 {
			return v[1 : end+1]
		}
	}
	if i := strings.Index(v, " #"); i >= 0 {
		v = v[:i]
	}
	return strings.TrimSpace(v)
}
