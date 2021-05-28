package main

import (
	"log"
	"os"
	"reflect"
	"strconv"

	"github.com/teamhephy/router/model"
	"github.com/teamhephy/router/nginx"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
)

func main() {
	nginx.Start()
	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to create config: %v", err)
	}
	if qps, err := strconv.ParseFloat(os.Getenv("RATE_LIMIT_QPS"), 32); err == nil {
		cfg.QPS = float32(qps)
		log.Printf("INFO: Setting QPS %f\n", qps)
	}
	if burst, err := strconv.Atoi(os.Getenv("RATE_LIMIT_BURST")); err == nil {
		cfg.Burst = burst
		log.Printf("INFO: Setting Burst %d\n", burst)
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to create client: %v.", err)
	}
	rateLimiter := flowcontrol.NewTokenBucketRateLimiter(0.1, 1)
	known := &model.RouterConfig{}
	// Main loop
	for {
		rateLimiter.Accept()
		routerConfig, err := model.Build(kubeClient)
		if err != nil {
			log.Printf("Error building model; not modifying certs or configuration: %v.", err)
			continue
		}
		if reflect.DeepEqual(routerConfig, known) {
			continue
		}
		log.Println("INFO: Router configuration has changed in k8s.")
		err = nginx.WriteCerts(routerConfig, "/opt/router/ssl")
		if err != nil {
			log.Printf("Failed to write certs; continuing with existing certs, dhparam, and configuration: %v", err)
			continue
		}
		err = nginx.WriteDHParam(routerConfig, "/opt/router/ssl")
		if err != nil {
			log.Printf("Failed to write dhparam; continuing with existing dhparam and configuration: %v", err)
			continue
		}
		err = nginx.WriteConfig(routerConfig, "/opt/router/conf/nginx.conf")
		if err != nil {
			log.Printf("Failed to write new nginx configuration; continuing with existing configuration: %v", err)
			continue
		}
		err = nginx.Reload()
		if err != nil {
			log.Printf("Failed to reload nginx; continuing with existing configuration: %v", err)
			continue
		}
		known = routerConfig
	}
}
