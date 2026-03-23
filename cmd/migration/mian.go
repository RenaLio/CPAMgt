package main

import (
	"context"
	"cpamgt/cmd/migration/wire"
	"cpamgt/internal/config"
	"cpamgt/internal/pkg/log"
	"flag"
	"fmt"
)

func main() {
	var envConf = flag.String("conf", "config/config.yaml", "config path, eg: -conf ./config/local.yml")
	flag.Parse()
	conf, err := config.NewConfig(*envConf)
	if err != nil {
		panic(err)
	}

	fmt.Printf("conf: %#v\n", conf)

	logger := log.NewLog(conf)

	app, cleanup, err := wire.NewWire(conf, logger)
	defer cleanup()
	if err != nil {
		panic(err)
	}
	logger.Info("migration start")
	if err = app.Run(context.Background()); err != nil {
		panic(err)
	}

}
