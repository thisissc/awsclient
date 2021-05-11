package main

import (
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"time"

	flag "github.com/spf13/pflag"
	"github.com/thisissc/awsclient"
	"github.com/thisissc/config"
	"github.com/thisissc/radixclient"
)

const (
	cacheKey = "TEST:AwsEcsServiceEndpoint:%s" // string
	cacheTTL = 5
)

var (
	wg sync.WaitGroup
)

type syncEcsSrvAddrsHandler struct {
	authority string
	endpoint  string
	cacheTTL  int64
}

func NewSyncEcsSrvAddrsHandler(authority, endpoint string, ttl int64) *syncEcsSrvAddrsHandler {
	return &syncEcsSrvAddrsHandler{
		authority: authority,
		endpoint:  endpoint,
		cacheTTL:  ttl,
	}
}

func (h *syncEcsSrvAddrsHandler) Handle() {
	sess := awsclient.GetSession()

	key := fmt.Sprintf(cacheKey, h.endpoint)

	t := time.NewTicker(time.Duration(h.cacheTTL/2) * time.Second)
	for {
		addrs := awsclient.GetEcsServiceAddrList(sess, h.authority, h.endpoint)
		err := radixclient.Save2RedisMutex(key, cacheTTL, addrs)
		if err != nil {
			log.Println(err)
		}
		<-t.C
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	help := false
	configFile := "config.toml"
	serviceList := []string{}

	flag.BoolVar(&help, "help", help, "Print this help")
	flag.StringVar(&configFile, "config", configFile, "Path of config file")
	flag.StringArrayVar(&serviceList, "service", serviceList, "Service name")
	flag.Parse()

	if help {
		flag.PrintDefaults()
		return
	}

	config.SetConfigFile(configFile)
	config.LoadConfig("AWS", &awsclient.Config{})
	config.LoadConfig("Radix", &radixclient.Config{})

	handlers := make([]*syncEcsSrvAddrsHandler, 0)
	for _, srvName := range serviceList {
		srv := NewSyncEcsSrvAddrsHandler("api02", srvName, cacheTTL)
		handlers = append(handlers, srv)
	}

	wg.Add(len(handlers))
	for _, handler := range handlers {
		go func(h *syncEcsSrvAddrsHandler) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Recovered %v\n", r)
					debug.PrintStack()
				}
			}()

			h.Handle()
		}(handler)
	}

	wg.Wait()
}
