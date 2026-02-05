package main

import (
	"fmt"
	"log"

	"example/service/internal/config"
)

//go:generate go run github.com/vovanwin/configgen/cmd/configgen --configs=./configs --output=./internal/config --package=config

func main() {
	// Создаем конфиг - загружает value.toml + config_{APP_ENV}.toml + config_local.toml
	cfg, err := config.NewConfig(config.Options{
		Dir: "./configs",
	})
	if err != nil {
		log.Fatalf("Ошибка загрузки конфига: %v", err)
	}

	fmt.Println("=== Конфигурация загружена ===")
	fmt.Printf("Окружение: %s\n", cfg.Env())
	fmt.Println()

	// Доступ через типизированные методы
	fmt.Println("--- Server ---")
	fmt.Printf("Host: %s\n", cfg.ServerHost())
	fmt.Printf("Port: %d\n", cfg.ServerPort())
	fmt.Printf("ReadTimeout: %v\n", cfg.ServerReadTimeout())
	fmt.Printf("WriteTimeout: %v\n", cfg.ServerWriteTimeout())
	fmt.Println()

	fmt.Println("--- Database ---")
	fmt.Printf("Host: %s:%d\n", cfg.DbHost(), cfg.DbPort())
	fmt.Printf("Name: %s\n", cfg.DbName())
	fmt.Printf("User: %s\n", cfg.DbUser())
	fmt.Printf("PoolSize: %d\n", cfg.DbPoolSize())
	fmt.Printf("MaxIdleTime: %v\n", cfg.DbMaxIdleTime())
	fmt.Println()

	fmt.Println("--- Log ---")
	fmt.Printf("Level: %s\n", cfg.LogLevel())
	fmt.Printf("Format: %s\n", cfg.LogFormat())
	fmt.Println()

	fmt.Println("--- App (from value.toml) ---")
	fmt.Printf("Name: %s\n", cfg.AppName())
	fmt.Printf("Version: %s\n", cfg.AppVersion())
	fmt.Println()

	fmt.Println("--- Limits ---")
	fmt.Printf("MaxConnections: %d\n", cfg.LimitsMaxConnections())
	fmt.Printf("MaxRequestSize: %d\n", cfg.LimitsMaxRequestSize())
	fmt.Printf("RequestTimeout: %v\n", cfg.LimitsRequestTimeout())
	fmt.Println()

	// Проверка окружения
	if cfg.IsProd() {
		fmt.Println("Режим: PRODUCTION")
	} else if cfg.IsDev() {
		fmt.Println("Режим: DEVELOPMENT")
	}

	// Доступ по ключу (для RTC в будущем)
	fmt.Println()
	fmt.Println("--- Доступ по ключу ---")
	fmt.Printf("Key %s = %v\n", config.ServerHost, cfg.GetValue(config.ServerHost))
	fmt.Printf("Key %s = %v\n", config.DbPoolSize, cfg.GetValue(config.DbPoolSize))
}
