package main

//go:generate go run github.com/yourname/configgen/cmd/configgen \
//  --configs=../configs \
//  --output=./internal/config \
//  --package=config
import (
	"github.com/vovanwin/configgen/example/service/internal/config"
	"log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("cfg: %+v", cfg)
}
