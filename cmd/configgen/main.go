package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/vovanwin/configgen/internal/generator"
	"github.com/vovanwin/configgen/internal/parser"
	"github.com/vovanwin/configgen/internal/schema"
)

func main() {
	configsDir := flag.String("configs", "./configs", "directory with config_*.yaml files")
	outDir := flag.String("output", "./internal/config", "output directory for generated code")
	pkgName := flag.String("package", "config", "package name for generated code")
	prefix := flag.String("env-prefix", "APP_ENV", "env prefix variable to decide runtime env (unused in generator)")

	flag.Parse()

	// find config_*.yaml files
	files, err := filepath.Glob(filepath.Join(*configsDir, "config_*.yaml"))
	if err != nil {
		log.Fatalf("glob: %v", err)
	}
	if len(files) == 0 {
		log.Fatalf("no config_*.yaml files found in %s", *configsDir)
	}

	asts := make([]map[string]*schema.Field, 0, len(files))
	for _, f := range files {
		m, err := parser.ParseFile(f)
		if err != nil {
			log.Fatalf("parse %s: %v", f, err)
		}
		asts = append(asts, m)
	}

	// intersect all ASTs
	s := asts[0]
	for i := 1; i < len(asts); i++ {
		s = schema.Intersect(s, asts[i])
	}

	if len(s) == 0 {
		log.Fatalf("empty schema after intersection — no common fields found")
	}

	if err := generator.Generate(*outDir, *pkgName, s); err != nil {
		log.Fatalf("generate: %v", err)
	}

	fmt.Println("generated to", *outDir)
	fmt.Println("hint: add the generated package to your service and use the runtime loader template to load values at runtime")
	fmt.Println("env-prefix flag is kept for compatibility; runtime loader expects APP_ENV (or configure manually)")
	_ = prefix
	os.Exit(0)
}
