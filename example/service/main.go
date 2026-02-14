package main

import (
	"fmt"
	"log"

	"example/service/internal/config"
)

//go:generate go run github.com/vovanwin/configgen/cmd/configgen --configs=./configs --output=./internal/config --package=config

func main() {
	result, err := config.LoadWithFlags(&config.LoadOptions{
		ConfigDir: "./configs",
		EnableEnv: true, // Включаем env var override
	})
	if err != nil {
		log.Fatalf("Ошибка загрузки конфига: %v", err)
	}

	cfg := result.Config

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

	fmt.Println("--- OAuth (nested sections) ---")
	fmt.Printf("Github ClientId: %s\n", cfg.Oauth.Github.ClientId)
	fmt.Printf("Github RedirectUrl: %s\n", cfg.Oauth.Github.RedirectUrl)
	fmt.Printf("VK ClientId: %s\n", cfg.Oauth.Vk.ClientId)
	fmt.Printf("VK RedirectUrl: %s\n", cfg.Oauth.Vk.RedirectUrl)
	fmt.Println()

	fmt.Println("--- Environment Check ---")
	if config.IsProduction() {
		fmt.Println("Режим: PRODUCTION")
	} else if config.IsStg() {
		fmt.Println("Режим: STG")
	} else {
		fmt.Println("Режим: Local")
	}
	fmt.Println()

	// Feature Flags (загружены из LoadWithFlags, с override из [flags] секции)
	fmt.Println("--- Feature Flags ---")
	flags := result.Flags
	fmt.Printf("NewCatalogUi: %v\n", flags.NewCatalogUi())
	fmt.Printf("RateLimit: %d\n", flags.RateLimit())
	fmt.Printf("ScoreThreshold: %.2f\n", flags.ScoreThreshold())
	fmt.Printf("BannerText: %s\n", flags.BannerText())
	fmt.Printf("Environment: %s (enum)\n", flags.Environment())
	fmt.Printf("LogFormat: %s (enum)\n", flags.LogFormat())
}
