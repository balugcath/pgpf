package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/balugcath/pgpf/internal/pkg/config"
	"github.com/balugcath/pgpf/internal/pkg/failover"
	rtm "github.com/balugcath/pgpf/internal/pkg/runtime_metric"
	"github.com/balugcath/pgpf/internal/pkg/transport"
	"github.com/balugcath/pgpf/pkg/prom_wrap"
	log "github.com/sirupsen/logrus"
)

type opts struct {
	configFile  string
	etcdAddress string
	etcdKey     string
	debugLevel  bool
}

func main() {
	options := opts{}
	flag.StringVar(&options.configFile, "f", "", "config file")
	flag.StringVar(&options.etcdAddress, "e", "", "etcd address")
	flag.StringVar(&options.etcdKey, "k", "", "etcd key")
	flag.BoolVar(&options.debugLevel, "v", false, "set debug log level")
	flag.Parse()

	log.SetFormatter(&log.TextFormatter{DisableColors: true})
	log.SetLevel(log.InfoLevel)
	if options.debugLevel {
		log.SetLevel(log.DebugLevel)
	}

	ctx, cancel := context.WithCancel(context.Background())

	cfg, err := config.NewConfig(options.configFile, options.etcdAddress, options.etcdKey)
	if err != nil {
		log.Fatalln(err)
	}

	metric := promwrap.NewProm(cfg.PrometheusListenPort, "/metrics")
	if cfg.PrometheusListenPort != "" {
		go func() {
			log.Fatalln(metric.Start())
		}()
	}

	failover.NewFailover(cfg, &transport.PG{}, metric).Start(ctx)

	rtm.NewHostStatus(cfg, metric, &transport.PG{}).Start()

	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT)
	for {
		switch <-sigs {
		case syscall.SIGTERM, syscall.SIGINT:
			cancel()
		case syscall.SIGHUP:
			c, err := config.NewConfig(options.configFile, options.etcdAddress, options.etcdKey)
			if err != nil {
				log.Println(err)
				continue
			}
			log.Println("config reloaded")
			cfg.Lock()
			*cfg = *c
		}
	}
}
