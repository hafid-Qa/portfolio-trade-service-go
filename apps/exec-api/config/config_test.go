package config

import (
	"strings"
	"testing"
	"time"
)

func validAuthConfig() Config {
	return Config{AccessTokenDuration: 15 * time.Minute, LoginMaxAttempts: 5, LoginPeriod: 15 * time.Minute}
}

func TestValidateAuth_Valid(t *testing.T) {
	c := validAuthConfig()
	if err := c.validateAuth(); err != nil {
		t.Fatalf("validateAuth() error = %v, want nil", err)
	}
}

func TestValidateAuth_RejectsNonPositive(t *testing.T) {
	cases := map[string]struct {
		mutate func(*Config)
		want   string
	}{
		"zero max attempts":     {func(c *Config) { c.LoginMaxAttempts = 0 }, "LOGIN_MAX_ATTEMPTS"},
		"negative max attempts": {func(c *Config) { c.LoginMaxAttempts = -1 }, "LOGIN_MAX_ATTEMPTS"},
		"zero period":           {func(c *Config) { c.LoginPeriod = 0 }, "LOGIN_PERIOD"},
		"negative period":       {func(c *Config) { c.LoginPeriod = -time.Minute }, "LOGIN_PERIOD"},
		"zero token duration":   {func(c *Config) { c.AccessTokenDuration = 0 }, "ACCESS_TOKEN_DURATION"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := validAuthConfig()
			tc.mutate(&c)
			err := c.validateAuth()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("validateAuth() error = %v, want it to name %s", err, tc.want)
			}
		})
	}
}
