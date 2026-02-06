package server

import (
	"fmt"
	"net/http"

	"example/service/internal/config"
)

type Server struct {
	cfg *config.Config
	env config.Configurator
}

func New(cfg *config.Config, env config.Configurator) *Server {
	return &Server{cfg: cfg, env: env}
}

// LogLevel возвращает уровень логирования: в prod форсируем error, в stg — warn
func (s *Server) LogLevel() string {
	if s.env.IsProduction() {
		return "error"
	}
	if s.env.IsStg() {
		return "warn"
	}
	return s.cfg.Log.Level
}

// ListenAddr возвращает адрес
func (s *Server) ListenAddr() string {
	return fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
}

// HealthHandler возвращает хендлер для health check
func (s *Server) HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		env := s.env.GetEnv()
		body := fmt.Sprintf(`{"status":"ok","env":"%s"}`, env)

		if s.cfg.Features.EnableMetrics {
			body = fmt.Sprintf(`{"status":"ok","env":"%s","metrics":true}`, env)
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, body)
	}
}

// DSN возвращает строку подключения к БД
func (s *Server) DSN() string {
	db := s.cfg.Db
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?pool_size=%d",
		db.User, db.Password, db.Host, db.Port, db.Name, db.PoolSize)
}
