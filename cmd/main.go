package main

import (
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/alecthomas/kingpin/v2"
	"github.com/nukosuke/go-zendesk/zendesk"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	listenAddress = kingpin.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":9633").String()
	metricsPath   = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
)

type ticketCollector struct {
	mutex       sync.RWMutex
	client      *zendesk.Client
	ticketCount *prometheus.Desc
}

func newTicketCollector(client *zendesk.Client) *ticketCollector {
	return &ticketCollector{
		client: client,
		ticketCount: prometheus.NewDesc(
			"zendesk_ticket_count",
			"Number of tickets in Zendesk",
			nil, nil,
		),
	}
}

func (c *ticketCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.ticketCount
}

func (c *ticketCollector) Collect(ch chan<- prometheus.Metric) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Get ticket count
	count, err := c.client.SearchCount(nil)
	if err != nil {
		log.Printf("Error getting ticket count: %v", err)
		return
	}

	ch <- prometheus.MustNewConstMetric(
		c.ticketCount,
		prometheus.GaugeValue,
		float64(count),
	)
}

func main() {
	kingpin.Parse()

	// Get Zendesk credentials from environment variables
	zendeskDomain := os.Getenv("ZENDESK_DOMAIN")
	if zendeskDomain == "" {
		log.Fatal("Environment variable ZENDESK_DOMAIN must be set")
	}

	zendeskEmail := os.Getenv("ZENDESK_EMAIL")
	if zendeskEmail == "" {
		log.Fatal("Environment variable ZENDESK_EMAIL must be set")
	}

	zendeskToken := os.Getenv("ZENDESK_TOKEN")
	if zendeskToken == "" {
		log.Fatal("Environment variable ZENDESK_TOKEN must be set")
	}

	// Create Zendesk client
	client, err := zendesk.NewClient(nil)
	if err != nil {
		log.Fatal(err)
	}

	client.SetSubdomain(zendeskDomain)
	client.SetCredential(zendesk.NewAPITokenCredential(zendeskEmail, zendeskToken))

	// Create and register collector
	collector := newTicketCollector(client)
	prometheus.MustRegister(collector)

	// Setup HTTP server
	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>Zendesk Exporter</title></head>
			<body>
			<h1>Zendesk Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})

	log.Printf("Starting Zendesk exporter on %s", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
