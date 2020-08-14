package common

import (
	"io/ioutil"
	"regexp"
	"strings"
)

const (
	CONFIG_AUTH_USERNAME = "auth_username"
	CONFIG_AUTH_PASSWORD = "auth_password"
)

type Config struct {
	m map[string]string
	e error
}

func ReadDefaultConfigFile(fileName string) Config {
	config := Config{
		m: make(map[string]string),
	}
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		config.e = err
	} else {
		config.readConfigFileImpl(data)
	}
	return config
}

func (c *Config) Get(key string) string {
	value, ok := c.m[key]
	if ok {
		return value
	}
	return ""
}

func (c *Config) GetBoolean(key string, def bool) bool {
	value, ok := c.m[key]
	if ok {
		return strings.EqualFold(value, "true") ||
			strings.EqualFold(value, "yes")
	}
	return def
}

func (c *Config) GetOrDefault(key string, def string) string {
	value, ok := c.m[key]
	if ok {
		return value
	}
	return def
}

func (c *Config) GetError() error {
	return c.e
}

func (c *Config) readConfigFileImpl(data []byte) {
	expr := regexp.MustCompile(`^\s*([a-zA-Z_]+):?\s+(\S+)\s*$`)

	for _, l := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "#") {
			continue
		}

		if m := expr.FindStringSubmatch(l); len(m) == 3 {
			key := m[1]
			value := m[2]

			c.m[key] = value
		}
	}
}
