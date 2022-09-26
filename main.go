package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	mp "github.com/mackerelio/go-mackerel-plugin"
	"github.com/namsral/flag"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	version string
)

const (
	endpoint = "https://api.sendgrid.com/v3/stats"
)

func userAgent() string {
	return fmt.Sprintf("mackerel-plugin-sendgrid/%s", version)
}

type SendgridStats struct {
	Stats []SendgridStat `json:"stats,omitempty"`
}

type SendgridStat struct {
	Metrics map[string]int `json:"metrics,omitempty"`
}

type SendgridPlugin struct {
	Prefix         string
	SendgridAPIKey string
}

func (s SendgridPlugin) FetchMetrics() (map[string]float64, error) {
	client := &http.Client{}

	api, err := url.Parse(endpoint)
	if err != nil {
		log.Printf("failed to parse endpoint: %s", endpoint)
		return nil, err
	}

	date := time.Now().Add(-24 * time.Hour)
	query := url.Values{}
	query.Add("start_date", date.Format("2006-01-02"))
	query.Add("end_date", date.Format("2006-01-02"))
	api.RawQuery = query.Encode()

	req, err := http.NewRequest("GET", api.String(), nil)
	if err != nil {
		log.Print("failed to new http request")
		return nil, err
	}

	req.Header.Set("user-agent", userAgent())
	req.Header.Set("authorization", fmt.Sprintf("Bearer %s", s.SendgridAPIKey))
	req.Header.Set("content-type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("failed to request Sendgrid stats API %s", api)
		return nil, err
	}
	defer resp.Body.Close()

	if status := resp.StatusCode; status != http.StatusOK {
		log.Print("unexpected http status")

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Print("failed to read response body")
			return nil, err
		}

		return nil, fmt.Errorf("Sendgrid returns status %d: %s", status, string(body))
	}

	body := new(bytes.Buffer)
	if _, err := io.Copy(body, resp.Body); err != nil {
		log.Print("failed to copy response body")
		return nil, err
	}

	stats := []SendgridStats{}
	if err := json.Unmarshal(body.Bytes(), &stats); err != nil {
		log.Printf("failed to unmarshal response body: %s", body.String())
		return nil, err
	}

	if len(stats) == 0 || len(stats[0].Stats) == 0 {
		log.Print("no stats found")
		return nil, nil
	}

	metrics := map[string]float64{}

	for k, v := range stats[0].Stats[0].Metrics {
		metrics[k] = float64(v)
	}

	return metrics, nil
}

func (s SendgridPlugin) GraphDefinition() map[string]mp.Graphs {
	prefix := cases.Title(language.English).String(s.MetricKeyPrefix())

	return map[string]mp.Graphs{
		"global": {
			Label: prefix,
			Unit:  mp.UnitInteger,
			Metrics: []mp.Metrics{
				{Name: "bounce_drops", Label: "BounceDrops"},
				{Name: "bounces", Label: "Bounces"},
				{Name: "clicks", Label: "Clicks"},
				{Name: "deferred", Label: "Deferred"},
				{Name: "delivered", Label: "Delivered"},
				{Name: "invalid_emails", Label: "InvalidEmails"},
				{Name: "opens", Label: "Opens"},
				{Name: "processed", Label: "Processed"},
				{Name: "requests", Label: "Requests"},
				{Name: "spam_report_drops", Label: "SpamReportDrops"},
				{Name: "spam_reports", Label: "SpamReports"},
				{Name: "unique_clicks", Label: "UniqueClicks"},
				{Name: "unique_opens", Label: "UniqueOpens"},
				{Name: "unsubscribe_drops", Label: "UnsubscribeDrops"},
				{Name: "unsubscribes", Label: "Unsubscribes"},
			},
		},
	}
}

func (s SendgridPlugin) MetricKeyPrefix() string {
	return s.Prefix
}

func main() {
	optPrefix := flag.String("metric-key-prefix", "sendgrid", "Metric key prefix")
	optSendgridAPIKey := flag.String("sendgrid-apikey", "", "API key of Sendgrid (needs access permission to get Stats)")
	flag.Parse()

	s := SendgridPlugin{
		Prefix:         *optPrefix,
		SendgridAPIKey: *optSendgridAPIKey,
	}

	plugin := mp.NewMackerelPlugin(s)
	plugin.Run()
}
