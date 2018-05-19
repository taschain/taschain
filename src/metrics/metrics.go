package metrics

import (
	"time"
	"github.com/rcrowley/go-metrics"
	"github.com/vrischmann/go-metrics-influxdb"
	"strings"
	"bytes"
	"fmt"
	"log"
)


var startTime time.Time
var Enabled bool

func init() {
	startTime = time.Now()
}

func EnableMetrics(enableInfluxdb bool) {
	Enabled = true
	go CollectTotalRunDurantion(time.Second*5)
	if enableInfluxdb {
		go EnableInfluxdbReport()
	}
}

func CollectTotalRunDurantion(duration time.Duration) {
	timer := metrics.GetOrRegisterTimer("total", metrics.DefaultRegistry)
	for {
		timer.UpdateSince(startTime)
		time.Sleep(duration)
	}
}

type DurationCollector struct {
	startTime time.Time
	module string
}

func (d *DurationCollector) Reset() {
	if !Enabled {
		return
	}
	d.startTime = time.Now()
}

func (d *DurationCollector) Stop() {
	if !Enabled {
		return
	}
	meter := metrics.GetOrRegisterMeter("module/" + d.module, metrics.DefaultRegistry)
	meter.Mark(int64(time.Since(d.startTime)))
}

func NewDurationCollector(module string) *DurationCollector{
	dc := DurationCollector{time.Now(), module}
	return &dc
}


func CollectRunDuration(module string) func(){
	if !Enabled {
		return func() {
		}
	}
	start := time.Now()
	return func() {
		meter := metrics.GetOrRegisterMeter("module/" + module, metrics.DefaultRegistry)
		meter.Mark(int64(time.Since(start)))
	}
}

func EnableInfluxdbReport() {
	go influxdb.InfluxDB(
		metrics.DefaultRegistry, // metrics registry
		time.Second * 5,        // interval
		"http://localhost:8086", // the InfluxDB url
		"gtas",                  // your InfluxDB database
		"",                // your InfluxDB user
		"",            // your InfluxDB password
	)
}

func ConsoleReport() {
	metricsMaps := metrics.DefaultRegistry.GetAll()
	var buffer bytes.Buffer
	buffer.WriteString("metrics:\n/************************************************************************************************/\n")
	total := metricsMaps["total"]["max"].(int64)
	for k, v := range metricsMaps {
		if strings.HasPrefix(k, "module") {
			buffer.WriteString(fmt.Sprintf("%s: %ds %.2f%%\n", k, v["count"].(int64)/int64(time.Second), float64(v["count"].(int64))/float64(total)*100))
		}
	}
	buffer.WriteString(fmt.Sprintf("total: %ds\n",  total/int64(time.Second)))
	buffer.WriteString("/************************************************************************************************/")
	log.Println(buffer.String())
}

