package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/Mautu/eureka_exporter/collector"

	"github.com/Mautu/eureka_exporter/conf"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	plog "github.com/prometheus/common/log"
)

var (
	configpath string
	config     *conf.Config
)

func init() {
	var err error
	flag.StringVar(&configpath, "conf", "./config.yml", "config file path")
	flag.Parse()
	if strings.Compare(configpath, "") == 0 {
		flag.Usage()
		plog.Infoln("config file path not set")
	}
	log.Println("config file is:", configpath)
	config, err = conf.LoadFile(configpath)
	if err != nil {
		plog.Fatalln("Fail to parse 'config.yml':", err)
	} else {
		plog.Infoln("load config success", config.Version, config.Port)
	}
}
func main() {
	metrics := collector.NewMetrics(config)
	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics)
	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
	        <head><title>eureka_exporter</title></head>
	        <body>
	        <h1>check_exporter</h1>
			<p>version: ` + config.Version + `</a></p>
	        <p><a href='metrics'>Metrics</a></p>
	        </body>
	        </html>`))
	})
	plog.Infoln("Listening on", ":"+strconv.Itoa(config.Port))
	plog.Fatal(http.ListenAndServe(":"+strconv.Itoa(config.Port), nil))
}
