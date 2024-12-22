package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/nsxbet/zendesk_exporter/internal/collector"

	"github.com/alecthomas/kingpin/v2"
	"github.com/nukosuke/go-zendesk/zendesk"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	listenAddress = kingpin.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":9101").String()
	metricsPath   = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
)

func getEnvOrFatal(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Environment variable %s must be set", key)
	}
	return value
}

func main() {
	kingpin.Parse()

	// Get Zendesk credentials from environment variables
	zendeskDomain := getEnvOrFatal("ZENDESK_DOMAIN")
	zendeskEmail := getEnvOrFatal("ZENDESK_EMAIL")
	zendeskAPIToken := getEnvOrFatal("ZENDESK_API_TOKEN")

	zendeskClient := newZendeskClient(
		zendeskDomain,
		zendeskEmail,
		zendeskAPIToken,
	)

	// Create and register collectors
	allTimeCollector := collector.NewAllTimeTicketsCollector(zendeskClient)
	recentCollector := collector.NewRecentTicketsCollector(zendeskClient)
	tagsCollector := collector.NewTagsTicketsCollector(zendeskClient)
	customFieldsCollector := collector.NewCustomFieldsCollector(zendeskClient)
	ticketsCollector := collector.NewTicketsCollector(zendeskClient)
	prometheus.MustRegister(allTimeCollector)
	prometheus.MustRegister(recentCollector)
	prometheus.MustRegister(tagsCollector)
	prometheus.MustRegister(customFieldsCollector)
	prometheus.MustRegister(ticketsCollector)

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

func newZendeskClient(domain, email, apiToken string) *zendesk.Client {
	if domain == "" || email == "" || apiToken == "" {
		log.Fatalf("Missing required environment variables")
	}

	client, err := zendesk.NewClient(&http.Client{Timeout: time.Second * 30})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	client.SetSubdomain(domain)
	client.SetCredential(zendesk.NewAPITokenCredential(email, apiToken))

	return client
}
