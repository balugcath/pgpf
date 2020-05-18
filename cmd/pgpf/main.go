package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/balugcath/pgpf/internal/pkg/config"
	"github.com/balugcath/pgpf/internal/pkg/failover"
)

type opts struct {
	configFile  string
	etcdAddress string
	etcdKey     string
}

func main() {
	options := opts{}
	flag.StringVar(&options.configFile, "f", "", "config file")
	flag.StringVar(&options.etcdAddress, "e", "http://192.168.1.16:2379", "etcd address")
	flag.StringVar(&options.etcdKey, "k", "pgpf", "etcd key")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())

	cfg, err := config.NewConfig(options.configFile, options.etcdAddress, options.etcdKey)
	if err != nil {
		log.Fatalln(err)
	}

	failover.NewFailover(cfg).Start(ctx)

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
