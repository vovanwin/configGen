package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/vovanwin/configgen/generator"
)

func main() {
	opts := generator.DefaultOptions()

	flag.StringVar(&opts.ConfigDir, "configs", opts.ConfigDir, "директория с конфиг файлами")
	flag.StringVar(&opts.OutputDir, "output", opts.OutputDir, "директория для генерации")
	flag.StringVar(&opts.Package, "package", opts.Package, "имя пакета")
	flag.Parse()

	fmt.Println("ConfigGen - генератор типобезопасных конфигов")
	fmt.Println()
	fmt.Printf("Конфиги: %s\n", opts.ConfigDir)
	fmt.Printf("Вывод:   %s\n", opts.OutputDir)
	fmt.Printf("Пакет:   %s\n", opts.Package)
	fmt.Println()
	fmt.Println("Парсинг файлов:")

	if err := generator.Generate(opts); err != nil {
		fmt.Fprintf(os.Stderr, "\n✗ Ошибка: %v\n", err)
		os.Exit(1)
	}
}
