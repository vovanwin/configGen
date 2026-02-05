package main

import (
	"fmt"
	"log"

	"example/service/internal/config"
)

//go:generate go run github.com/vovanwin/configgen/cmd/configgen --configs=./configs --output=./internal/config --package=config

func main() {
	// Загрузка конфига на основе переменной окружения APP_ENV
	// Порядок: value.toml -> config_{env}.toml -> config_local.toml (если есть)
	cfg, err := config.Load(&config.LoadOptions{
		ConfigDir:           "./configs",
		EnableLocalOverride: true,
	})
	if err != nil {
		log.Fatalf("Ошибка загрузки конфига: %v", err)
	}

	fmt.Println("=== Конфигурация загружена ===")
	fmt.Printf("Окружение: %s\n", config.GetEnv())
	fmt.Println()

	// Доступ к значениям сервера
	fmt.Println("--- Server ---")
	fmt.Printf("Host: %s\n", cfg.Server.Host)
	fmt.Printf("Port: %d\n", cfg.Server.Port)
	fmt.Printf("ReadTimeout: %v\n", cfg.Server.ReadTimeout)
	fmt.Printf("WriteTimeout: %v\n", cfg.Server.WriteTimeout)
	fmt.Println()

	// Доступ к значениям БД
	fmt.Println("--- Database ---")
	fmt.Printf("Host: %s:%d\n", cfg.Db.Host, cfg.Db.Port)
	fmt.Printf("Name: %s\n", cfg.Db.Name)
	fmt.Printf("User: %s\n", cfg.Db.User)
	fmt.Printf("PoolSize: %d\n", cfg.Db.PoolSize)
	fmt.Printf("MaxIdleTime: %v\n", cfg.Db.MaxIdleTime)
	fmt.Println()

	// Доступ к логам
	fmt.Println("--- Log ---")
	fmt.Printf("Level: %s\n", cfg.Log.Level)
	fmt.Printf("Format: %s\n", cfg.Log.Format)
	fmt.Println()

	// Доступ к константам из value.toml
	values := config.GetValue()
	if values != nil {
		fmt.Println("--- Constants (value.toml) ---")
		fmt.Printf("App: %s v%s\n", values.App.Name, values.App.Version)
		fmt.Printf("MaxConnections: %d\n", values.Limits.MaxConnections)
		fmt.Printf("RequestTimeout: %v\n", values.Limits.RequestTimeout)
		fmt.Printf("EnableMetrics: %v\n", values.Features.EnableMetrics)
		fmt.Printf("EnableTracing: %v\n", values.Features.EnableTracing)
		fmt.Println()
	}

	// Проверка окружения
	fmt.Println("--- Environment Check ---")
	if config.IsProduction() {
		fmt.Println("Режим: PRODUCTION")
	} else if config.IsStg() {
		fmt.Println("Режим: STG")
	} else {
		fmt.Println("Режим: Local")
	}
}
