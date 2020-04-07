package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// JhopkinsData - https://corona.lmao.ninja/jhucsse
type JhopkinsData []struct {
	Country   string `json:"country"`
	Province  string `json:"province"`
	City      string `json:"city"`
	UpdatedAt string `json:"updatedAt"`
	Stats     struct {
		Confirmed string  `json:"confirmed"`
		Deaths    string  `json:"deaths"`
		Recovered float64 `json:"recovered"`
	} `json:"stats"`
	Coordinates struct {
		Latitude  string `json:"latitude"`
		Longitude string `json:"longitude"`
	} `json:"coordinates"`
}

// Metrics endpoint
const (
	BaseURL   = "https://corona.lmao.ninja/jhucsse"
	UserAgent = "Mozilla/4.0 (compatible; MSIE 8.0; Windows NT 6.0)"
	namespace = "covid19JHP"
)

// Prometheus server listen address
var addr = flag.String("listen-address", ":9679", "Prometheus exporter will use this address")

// Exporter structs
type Exporter struct {
	sync.Mutex

	JHcases  *prometheus.GaugeVec
	JHdeaths *prometheus.GaugeVec
}

// NewExporter func
func NewExporter() *Exporter {
	return &Exporter{

		JHcases: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "confirmed_cases",
				Help:      "John Hopkins data confirmed cases",
			},
			[]string{"country", "province", "city", "latitude", "longitude"},
		),
		JHdeaths: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "deaths",
				Help:      "John Hopkins data confirmed deaths",
			},
			[]string{"country", "province", "city", "latitude", "longitude"},
		),
	}
}

// Describe implements the prometheus.Collector interface.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	// John Hopkins Data
	e.JHcases.Describe(ch)
	e.JHdeaths.Describe(ch)

}

// Collect method
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.Lock()
	defer e.Unlock()

	// http request
	JHreq, err := http.NewRequest("GET", BaseURL, nil)
	if err != nil {
		fmt.Print(err.Error())
	}
	// Setting basic headers
	JHreq.Header.Set("User-Agent", "Covid19 stats prometheus exporter")
	JHresp, err := http.DefaultClient.Do(JHreq)
	if err != nil {
		fmt.Print(err.Error())
	}
	defer JHresp.Body.Close()
	JHbody, readErr := ioutil.ReadAll(JHresp.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	JHdata := JhopkinsData{}
	json.Unmarshal([]byte(JHbody), &JHdata)

	for k := 0; k < len(JHdata); k++ {
		// s as stats variable
		x := JHdata[k]
		confirm, err := strconv.ParseFloat(x.Stats.Confirmed, 64)
		if err != nil {
			log.Fatal(err)
		}
		deaths, err := strconv.ParseFloat(x.Stats.Deaths, 64)
		if err != nil {
			log.Fatal(err)
		}
		e.JHcases.WithLabelValues(
			x.Country,
			x.Province,
			x.City,
			x.Coordinates.Latitude,
			x.Coordinates.Longitude).Set(confirm)
		e.JHdeaths.WithLabelValues(
			x.Country,
			x.Province,
			x.City,
			x.Coordinates.Latitude,
			x.Coordinates.Longitude).Set(deaths)

	}

	e.JHcases.Collect(ch)
	e.JHdeaths.Collect(ch)

}

func init() {
	// Unregister all metrics from go_*
	prometheus.Unregister(prometheus.NewGoCollector())
	// Register all metrics defined on NewExporter
	prometheus.MustRegister(NewExporter())

}

func main() {

	// setting up metrics endpoint for Prometheus
	flag.Parse()
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Covid19 Data Prometheus Exporter from John Hopkins data</title></head>
             <body>
			 <h2>Covid19 data Prometheus Exporter</h2>
			 Why?  why not,
             <p><a href='/metrics'>Metrics</a></p>
             </body>
             </html>`))
	})
	log.Fatal(http.ListenAndServe(*addr, nil))

}
