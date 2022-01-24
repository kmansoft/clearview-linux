package common

import (
	"io/ioutil"
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
	for _, l := range strings.Split(string(data), "\n") {
		t := strings.TrimSpace(l)

		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}

		s := strings.IndexAny(t, ":=")
		if s < 0 {
			s = strings.IndexByte(t, ' ')
		}
		if s > 0 {
			key := strings.TrimSpace(t[:s])
			value := strings.TrimSpace(t[s+1:])

			i := len(value)
			if i > 0 && value[0] == '"' && value[i-1] == '"' {
				value = value[1 : i-1]
			}

			c.m[key] = value
		}
	}
}
