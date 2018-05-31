package metrics

import (
	"time"
	"github.com/rcrowley/go-metrics"
	"github.com/vrischmann/go-metrics-influxdb"
	"strings"
	"bytes"
	"fmt"
	"log"
	"os"
	"net/http"
)


var Enabled bool

const MetricsEnabledFlag  = "metrics"
const DashboardEnabledFlag  = "dashboard"

func init() {
	for _, arg := range os.Args {
		if flag := strings.TrimLeft(arg, "-"); flag == MetricsEnabledFlag || flag == DashboardEnabledFlag {
			if flag == MetricsEnabledFlag {
				EnableMetrics(false)
				http.Handle("/metrics", http.HandlerFunc(reportHandler))
				go http.ListenAndServe(":9999", nil)
				log.Println("metrics serving on http://127.0.0.1:9999")
			}
			if flag == DashboardEnabledFlag {
				EnableMetrics(true)
			}
		}
	}
}

func EnableMetrics(enableInfluxdb bool) {
	Enabled = true
	go collectTotalRunDurantion(time.Second)
	if enableInfluxdb {
		go EnableInfluxdbReport()
	}
}

func collectTotalRunDurantion(duration time.Duration) {
	totlaMeter := metrics.GetOrRegisterMeter("total", metrics.DefaultRegistry)
	for {
		time.Sleep(duration)
		totlaMeter.Mark(int64(duration))
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
	log.Println(getReport())
}

func getReport() string {
	metricsMaps := metrics.DefaultRegistry.GetAll()
	var buffer bytes.Buffer
	buffer.WriteString("metrics:\n/************************************************************************************************/\n")
	total := metricsMaps["total"]["count"].(int64)
	for k, v := range metricsMaps {
		if strings.HasPrefix(k, "module") {
			buffer.WriteString(fmt.Sprintf("%s: %ds %.2f%% \n", k, v["count"].(int64)/int64(time.Second), float64(v["count"].(int64))/float64(total)*100))
		}
	}
	buffer.WriteString(fmt.Sprintf("total: %ds\n",  total/int64(time.Second)))
	buffer.WriteString("/************************************************************************************************/")
	log.Println(buffer.String())
	return buffer.String()
}

func reportHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html;charset=utf-8")
	report := fmt.Sprintf("<html><body>%s</body></html>", strings.Replace(getReport(), "\n", "<br>", -1))
	fmt.Fprintln(w, report)
}

