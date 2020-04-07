package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Covid19MX struct
type Covid19MX []struct {
	CaseID            float64 `json:"n0_caso"`
	State             string  `json:"estado"`
	Sex               string  `json:"sexo"`
	Age               float64 `json:"edad"`
	DateSintomStarted string  `json:"fecha_de_inicio_de_sintomas"`
	IDnRtSecDna       string  `json:"identificacion_de_covid_19_por_rt_pcrsecuencia_de_dna"`
	ArrivedFrom       string  `json:"procedencia"`
	EntryToMX         string  `json:"fecha_del_llegada_a_mexico"`
}

// Metrics endpoint
const (
	BaseURL   = "https://bridge.buddyweb.fr/api/covd19mx/"
	UserAgent = "Mozilla/4.0 (compatible; MSIE 8.0; Windows NT 6.0)"
	date      = "latest"
	baseURLMX = BaseURL + date
	namespace = "covid19MX"
)

// Prometheus server listen address
var addr = flag.String("listen-address", ":9677", "Prometheus exporter will use this address")

// Exporter structs
type Exporter struct {
	CaseID *prometheus.GaugeVec
	Age    *prometheus.GaugeVec
}

// NewExporter func
func NewExporter() *Exporter {
	return &Exporter{
		CaseID: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "caseID",
				Help:      "CaseID",
			},
			[]string{"state", "sex", "date_sintoms_started", "arrived_from", "entry_to_mx_date"},
		),

		Age: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "Age",
				Help:      "Person Age",
			},
			[]string{"state", "sex", "date_sintoms_started", "arrived_from", "entry_to_mx_date"},
		),
	}
}

// Describe implements the prometheus.Collector interface.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.CaseID.Describe(ch)
	e.Age.Describe(ch)
}

// Collect method
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	//e.Lock()
	//defer e.Unlock()

	// http request
	req, err := http.NewRequest("GET", baseURLMX, nil)
	if err != nil {
		fmt.Print(err.Error())
	}
	// Setting basic headers
	req.Header.Set("User-Agent", "Covid19MX stats prometheus exporter")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Print(err.Error())
	}
	defer resp.Body.Close()
	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}
	MXData := Covid19MX{}
	json.Unmarshal([]byte(body), &MXData)

	for cases := 0; cases < len(MXData); cases++ {
		d := MXData[cases]
		e.CaseID.WithLabelValues(
			d.State, d.Sex, d.DateSintomStarted, d.ArrivedFrom, d.EntryToMX).Set(d.CaseID)
		e.Age.WithLabelValues(
			d.State, d.Sex, d.DateSintomStarted, d.ArrivedFrom, d.EntryToMX).Set(d.Age)
	}

	e.CaseID.Collect(ch)
	e.Age.Collect(ch)
}

func init() {
	// Unregister all metrics from go_*
	prometheus.Unregister(prometheus.NewGoCollector())

	// Register all metrics defined on NewExporter
	prometheus.MustRegister(NewExporter())

}

func main() {

	// Setting up metrics endpoint for Prometheus
	flag.Parse()
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Covid19 Mexico only data Prometheus Exporter</title></head>
             <body>
			 <h2>Covid19 Mexico only data - Prometheus Exporter</h2>
			 Why?  why not,
             <p><a href='/metrics'>Metrics</a></p>
             </body>
             </html>`))
	})

	log.Fatal(http.ListenAndServe(*addr, nil))

}
