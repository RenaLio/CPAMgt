package main

import (
	"context"
	"cpamgt/cmd/server/wire"
	"cpamgt/internal/config"
	"cpamgt/pkg/log"
	"flag"
	"fmt"
	"log/slog"
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
	logger.Info("server start", slog.String("host", fmt.Sprintf("http://%s:%d", conf.Http.Host, conf.Http.Port)))
	logger.Info("docs addr", slog.String("addr", fmt.Sprintf("http://%s:%d/swagger/index.html", conf.Http.Host, conf.Http.Port)))
	if err = app.Run(context.Background()); err != nil {
		panic(err)
	}

}
