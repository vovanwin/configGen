package main

import (
	"fmt"
	"log"

	"example/service/internal/config"
)

//go:generate go run github.com/vovanwin/configgen/cmd/configgen --configs=./configs --output=./internal/config --package=config

func main() {
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

	fmt.Println("--- Server ---")
	fmt.Printf("Host: %s\n", cfg.Server.Host)
	fmt.Printf("Port: %d\n", cfg.Server.Port)
	fmt.Printf("ReadTimeout: %v\n", cfg.Server.ReadTimeout)
	fmt.Printf("WriteTimeout: %v\n", cfg.Server.WriteTimeout)
	fmt.Println()

	fmt.Println("--- Database ---")
	fmt.Printf("Host: %s:%d\n", cfg.Db.Host, cfg.Db.Port)
	fmt.Printf("Name: %s\n", cfg.Db.Name)
	fmt.Printf("User: %s\n", cfg.Db.User)
	fmt.Printf("PoolSize: %d\n", cfg.Db.PoolSize)
	fmt.Printf("MaxIdleTime: %v\n", cfg.Db.MaxIdleTime)
	fmt.Println()

	fmt.Println("--- Log ---")
	fmt.Printf("Level: %s\n", cfg.Log.Level)
	fmt.Printf("Format: %s\n", cfg.Log.Format)
	fmt.Println()

	// Константы из value.toml — тоже в cfg (всё мержится в одну структуру)
	fmt.Println("--- Constants (from value.toml) ---")
	fmt.Printf("App: %s v%s\n", cfg.App.Name, cfg.App.Version)
	fmt.Printf("MaxConnections: %d\n", cfg.Limits.MaxConnections)
	fmt.Printf("RequestTimeout: %v\n", cfg.Limits.RequestTimeout)
	fmt.Printf("EnableMetrics: %v\n", cfg.Features.EnableMetrics)
	fmt.Printf("EnableTracing: %v\n", cfg.Features.EnableTracing)
	fmt.Println()

	fmt.Println("--- Environment Check ---")
	if config.IsProduction() {
		fmt.Println("Режим: PRODUCTION")
	} else if config.IsStg() {
		fmt.Println("Режим: STG")
	} else {
		fmt.Println("Режим: Local")
	}
}
