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
	"github.com/vovanwin/configgen/pkg/types"
)

func main() {
	configsDir := flag.String("configs", "./configs", "directory with config files")
	outDir := flag.String("output", "./internal/config", "output directory for generated code")
	pkgName := flag.String("package", "config", "package name for generated code")
	envPrefix := flag.String("env-prefix", "APP_ENV", "env variable name for environment detection")
	withLoader := flag.Bool("with-loader", true, "generate loader.gen.go for runtime loading")
	withVault := flag.Bool("with-vault", false, "include Vault integration placeholders")
	withRTC := flag.Bool("with-rtc", false, "include RTC (real-time config) placeholders")
	mode := flag.String("mode", "intersect", "schema mode: intersect (common fields) or union (all fields)")

	flag.Parse()

	// Parse value.toml (constants) separately
	var valueFields map[string]*types.Field
	valuePath := filepath.Join(*configsDir, "value.toml")
	if _, err := os.Stat(valuePath); err == nil {
		valueFields, err = parser.ParseFile(valuePath)
		if err != nil {
			log.Fatalf("parse value.toml: %v", err)
		}
		fmt.Printf("parsed: value.toml (%d top-level fields)\n", len(valueFields))
	}

	// Find config_*.toml files (environment configs)
	envConfigs, err := filepath.Glob(filepath.Join(*configsDir, "config_*.toml"))
	if err != nil {
		log.Fatalf("glob: %v", err)
	}

	// Filter out config_local.toml from schema generation
	var configFiles []string
	for _, f := range envConfigs {
		if filepath.Base(f) != "config_local.toml" {
			configFiles = append(configFiles, f)
		}
	}

	if len(configFiles) == 0 && valueFields == nil {
		log.Fatalf("need at least value.toml or config_*.toml files in %s", *configsDir)
	}

	// Parse environment configs
	var envAsts []map[string]*types.Field
	for _, f := range configFiles {
		m, err := parser.ParseFile(f)
		if err != nil {
			log.Fatalf("parse %s: %v", f, err)
		}
		envAsts = append(envAsts, m)
		fmt.Printf("parsed: %s (%d top-level fields)\n", filepath.Base(f), len(m))
	}

	// Build schema for environment configs
	var envSchema map[string]*types.Field
	if len(envAsts) > 0 {
		switch *mode {
		case "intersect":
			envSchema = schema.Intersect(envAsts...)
		case "union":
			envSchema = schema.Union(envAsts...)
		default:
			log.Fatalf("unknown mode: %s (use 'intersect' or 'union')", *mode)
		}
	}

	// Merge value.toml fields with environment config fields
	var s map[string]*types.Field
	if valueFields != nil && envSchema != nil {
		s = schema.Union(valueFields, envSchema)
	} else if valueFields != nil {
		s = valueFields
	} else {
		s = envSchema
	}

	if len(s) == 0 {
		log.Fatalf("empty schema — no fields found")
	}

	// Generate code
	opts := generator.Options{
		OutputDir:   *outDir,
		PackageName: *pkgName,
		EnvPrefix:   *envPrefix,
		WithLoader:  *withLoader,
		WithVault:   *withVault,
		WithRTC:     *withRTC,
	}

	if err := generator.Generate(opts, s); err != nil {
		log.Fatalf("generate: %v", err)
	}

	fmt.Println()
	fmt.Println("✓ Generated files:")
	fmt.Printf("  - %s/config.gen.go\n", *outDir)
	if *withLoader {
		fmt.Printf("  - %s/loader.gen.go\n", *outDir)
	}
	fmt.Println()
	fmt.Println("Config files order (runtime):")
	fmt.Println("  1. value.toml      - base constants (optional)")
	fmt.Println("  2. config_{env}.toml - environment-specific")
	fmt.Println("  3. config_local.toml - local overrides (optional, don't commit)")
}
