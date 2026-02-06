package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"example/service/internal/config"
	"example/service/internal/config/mock"
)

// testConfig возвращает тестовую конфигурацию
func testConfig() *config.Config {
	return &config.Config{
		Env: config.EnvLocal,
		App: config.App{Name: "test-service", Version: "0.0.1"},
		Server: config.Server{
			Host:         "127.0.0.1",
			Port:         9090,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		},
		Db: config.Db{
			Host:     "localhost",
			Port:     5432,
			Name:     "testdb",
			User:     "testuser",
			Password: "testpass",
			PoolSize: 5,
		},
		Log: config.Log{
			Level:  "debug",
			Format: "text",
		},
		Features: config.Features{
			EnableMetrics: false,
			EnableTracing: false,
		},
		Limits: config.Limits{
			MaxConnections: 10,
			RequestTimeout: 3 * time.Second,
		},
	}
}

// --- Тесты с моком Configurator ---

func TestLogLevel_Production(t *testing.T) {
	cfgMock := mock.NewConfiguratorMock(t)
	cfgMock.IsProductionMock.Return(true)
	cfgMock.IsStgMock.Optional().Return(false)

	srv := New(testConfig(), cfgMock)

	got := srv.LogLevel()
	if got != "error" {
		t.Errorf("LogLevel() в prod = %q, ожидалось %q", got, "error")
	}
}

func TestLogLevel_Staging(t *testing.T) {
	cfgMock := mock.NewConfiguratorMock(t)
	cfgMock.IsProductionMock.Return(false)
	cfgMock.IsStgMock.Return(true)

	srv := New(testConfig(), cfgMock)

	got := srv.LogLevel()
	if got != "warn" {
		t.Errorf("LogLevel() в stg = %q, ожидалось %q", got, "warn")
	}
}

func TestLogLevel_Local(t *testing.T) {
	cfgMock := mock.NewConfiguratorMock(t)
	cfgMock.IsProductionMock.Return(false)
	cfgMock.IsStgMock.Return(false)

	cfg := testConfig()
	cfg.Log.Level = "debug"
	srv := New(cfg, cfgMock)

	got := srv.LogLevel()
	if got != "debug" {
		t.Errorf("LogLevel() в local = %q, ожидалось %q", got, "debug")
	}
}

// --- Тесты HTTP хендлера с моком ---

func TestHealthHandler_WithoutMetrics(t *testing.T) {
	cfgMock := mock.NewConfiguratorMock(t)
	cfgMock.GetEnvMock.Return("local")

	cfg := testConfig()
	cfg.Features.EnableMetrics = false
	srv := New(cfg, cfgMock)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	srv.HealthHandler().ServeHTTP(rec, req)

	resp := rec.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, ожидалось %d", resp.StatusCode, http.StatusOK)
	}

	expected := `{"status":"ok","env":"local"}`
	if string(body) != expected {
		t.Errorf("body = %s, ожидалось %s", body, expected)
	}
}

func TestHealthHandler_WithMetrics(t *testing.T) {
	cfgMock := mock.NewConfiguratorMock(t)
	cfgMock.GetEnvMock.Return("prod")

	cfg := testConfig()
	cfg.Features.EnableMetrics = true
	srv := New(cfg, cfgMock)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	srv.HealthHandler().ServeHTTP(rec, req)

	body, _ := io.ReadAll(rec.Result().Body)

	expected := `{"status":"ok","env":"prod","metrics":true}`
	if string(body) != expected {
		t.Errorf("body = %s, ожидалось %s", body, expected)
	}
}

// --- Тесты с конфигом напрямую (для интеграционных тестов с БД) ---

func TestDSN(t *testing.T) {
	cfg := testConfig()
	cfg.Db.Host = "testdb-host"
	cfg.Db.Port = 5433
	cfg.Db.Name = "integration_db"
	cfg.Db.User = "ci_user"
	cfg.Db.Password = "ci_pass"
	cfg.Db.PoolSize = 3

	cfgMock := mock.NewConfiguratorMock(t)

	srv := New(cfg, cfgMock)

	expected := "postgres://ci_user:ci_pass@testdb-host:5433/integration_db?pool_size=3"
	got := srv.DSN()
	if got != expected {
		t.Errorf("DSN() = %q, ожидалось %q", got, expected)
	}
}

func TestListenAddr(t *testing.T) {
	cfg := testConfig()
	cfg.Server.Host = "0.0.0.0"
	cfg.Server.Port = 8080

	cfgMock := mock.NewConfiguratorMock(t)

	srv := New(cfg, cfgMock)

	expected := "0.0.0.0:8080"
	got := srv.ListenAddr()
	if got != expected {
		t.Errorf("ListenAddr() = %q, ожидалось %q", got, expected)
	}
}
