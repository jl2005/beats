package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var namespace = "filebeat"
var promHandle http.Handler
var mutex sync.Mutex
var gaugesMap map[string]prometheus.Gauge
var counterMap map[string]prometheus.Counter
var preValue map[string]float64

var gauges = map[string]bool{
	"libbeat.pipeline.events.active": true,
	"libbeat.pipeline.clients":       true,
	"libbeat.config.module.running":  true,
	"registrar.states.current":       true,
	"filebeat.harvester.running":     true,
	"filebeat.harvester.open_files":  true,
	"beat.memstats.memory_total":     true,
	"beat.memstats.memory_alloc":     true,
	"beat.memstats.gc_next":          true,
	"beat.info.uptime.ms":            true,
	"beat.cpu.user.ticks":            true,
	"beat.cpu.user.time":             true,
	"beat.cpu.system.ticks":          true,
	"beat.cpu.system.time":           true,
	"beat.cpu.total.value":           true,
	"beat.cpu.total.ticks":           true,
	"beat.cpu.total.time":            true,
	"system.load.1":                  true,
	"system.load.5":                  true,
	"system.load.15":                 true,
	"system.load.norm.1":             true,
	"system.load.norm.5":             true,
	"system.load.norm.15":            true,
}

func init() {
	promHandle = promhttp.Handler()
	gaugesMap = make(map[string]prometheus.Gauge)
	counterMap = make(map[string]prometheus.Counter)
	preValue = make(map[string]float64)
}

// Start starts the metrics api endpoint on the configured host and port
func Start(cfg *common.Config) {
	cfgwarn.Experimental("Metrics endpoint is enabled.")
	config := DefaultConfig
	cfg.Unpack(&config)

	logp.Info("Starting stats endpoint")
	go func() {
		mux := http.NewServeMux()

		// register handlers
		mux.HandleFunc("/", rootHandler())
		mux.HandleFunc("/stats", statsHandler)
		mux.HandleFunc("/dataset", datasetHandler)
		mux.HandleFunc("/metrics", metricsHandler)

		url := config.Host + ":" + strconv.Itoa(config.Port)
		logp.Info("Metrics endpoint listening on: %s", url)
		endpoint := http.ListenAndServe(url, mux)
		logp.Info("finished starting stats endpoint: %v", endpoint)
	}()
}

func rootHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Return error page
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		data := monitoring.CollectStructSnapshot(monitoring.GetNamespace("state").GetRegistry(), monitoring.Full, false)

		print(w, data, r.URL)
	}
}

// statsHandler report expvar and all libbeat/monitoring metrics
func statsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	data := monitoring.CollectStructSnapshot(monitoring.GetNamespace("stats").GetRegistry(), monitoring.Full, false)

	print(w, data, r.URL)
}

func datasetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	data := monitoring.CollectStructSnapshot(monitoring.GetNamespace("dataset").GetRegistry(), monitoring.Full, false)

	print(w, data, r.URL)
}

func print(w http.ResponseWriter, data common.MapStr, u *url.URL) {
	query := u.Query()
	if _, ok := query["pretty"]; ok {
		fmt.Fprintf(w, data.StringToPrint())
	} else {
		fmt.Fprintf(w, data.String())
	}
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	snap := monitoring.CollectFlatSnapshot(nil, monitoring.Full, false)
	for key, value := range snap.Ints {
		if _, ok := gauges[key]; ok {
			setGauges(key, float64(value))
		} else {
			setCounter(key, float64(value))
		}
	}

	for key, value := range snap.Floats {
		if _, ok := gauges[key]; ok {
			setGauges(key, value)
		} else {
			setCounter(key, value)
		}
	}
	mutex.Unlock()

	promHandle.ServeHTTP(w, r)
}

func setGauges(name string, value float64) {
	//TODO add lock
	if v, ok := gaugesMap[name]; ok {
		v.Set(value)
	} else {
		g := prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      strings.Replace(name, ".", "_", -1),
			Help:      name,
		})
		gaugesMap[name] = g
		prometheus.MustRegister(g)
	}
}

func setCounter(name string, value float64) {
	if v, ok := counterMap[name]; ok {
		if pre, ok := preValue[name]; ok {
			if value-pre > 0 {
				v.Add(value - pre)
				preValue[name] = value
			}
		}
	} else {
		c := prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      strings.Replace(name, ".", "_", -1),
			Help:      name,
		})
		counterMap[name] = c
		prometheus.MustRegister(c)
		c.Add(value)
		preValue[name] = value
	}
}
