package config_test

import (
	"testing"

	"example/service/internal/config"
)

func TestEnvVarOverride(t *testing.T) {
	t.Setenv("APP_SERVER__HOST", "envhost.example.com")
	t.Setenv("APP_SERVER__PORT", "9999")
	t.Setenv("APP_DB__NAME", "test_db")

	cfg, err := config.Load(&config.LoadOptions{
		ConfigDir:   "../../configs",
		Environment: config.EnvLocal,
		EnableEnv:   true,
	})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Server.Host != "envhost.example.com" {
		t.Errorf("expected host from env, got %s", cfg.Server.Host)
	}

	if cfg.Server.Port != 9999 {
		t.Errorf("expected port from env, got %d", cfg.Server.Port)
	}

	if cfg.Db.Name != "test_db" {
		t.Errorf("expected db name from env, got %s", cfg.Db.Name)
	}
}

func TestEnvVarOverrideDisabled(t *testing.T) {
	t.Setenv("APP_SERVER__HOST", "should-not-override.com")

	cfg, err := config.Load(&config.LoadOptions{
		ConfigDir:   "../../configs",
		Environment: config.EnvLocal,
		EnableEnv:   false,
	})
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Server.Host == "should-not-override.com" {
		t.Error("env var should not override when EnableEnv=false")
	}

	if cfg.Server.Host != "localhost" {
		t.Errorf("expected value from config file, got %s", cfg.Server.Host)
	}
}
