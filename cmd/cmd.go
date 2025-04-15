package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"impala-exporter/config"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

var (
	nodeConfigs []nodeConfig
	metric      = "java.lang:type=GarbageCollector,name=PS MarkSweep"
	uriJmx      = "/jmx"
)

type impalaExporter struct{}

type JmxResponse struct {
	Beans []map[string]interface{} `json:"beans"`
}

type nodeConfig struct {
	ip     string
	jmxUrl string
}

type metricInfo struct {
	Desc *prometheus.Desc
	Type prometheus.ValueType
}

var listMetric = []string{
	"totalCommitedAfterGc",
	"totalInitAfterGc",
	"totalUsedAfterGc",
	"totalMaxAfterGc",
	"totalCommitedBeforeGc",
	"totalInitBeforeGc",
	"totalUsedBeforeGc",
	"totalMaxBeforeGc",
	"duration",
}

func (e *impalaExporter) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range listMetric {
		metricProm := newServerMetric(m, "JMX metric "+m, prometheus.GaugeValue)
		ch <- metricProm.Desc
	}
}

func (e *impalaExporter) Collect(ch chan<- prometheus.Metric) {
	collectionMetrics(ch)
}

func newServerMetric(metricName string, docString string, t prometheus.ValueType) metricInfo {
	return metricInfo{
		Desc: prometheus.NewDesc(
			prometheus.BuildFQName("impala", "jmx", metricName),
			docString,
			[]string{"ip"},
			prometheus.Labels{"cluster": "impala"},
		),
		Type: t,
	}
}

func convertSliceToMap(beans []map[string]interface{}) map[string]interface{} {
	temp := make(map[string]interface{})
	for _, bean := range beans {
		name := bean["name"].(string)
		temp[name] = bean
	}
	return temp
}

// FetchJmxData fetches JMX data from the given URL
func fetchJmxData(url string) (JmxResponse, error) {
	client := resty.New()
	resp, err := client.R().Get(url)
	if err != nil {
		return JmxResponse{}, err
	}
	if resp.StatusCode() != http.StatusOK {
		return JmxResponse{}, fmt.Errorf("status code error: %d %s", resp.StatusCode(), resp.String())
	}
	var data JmxResponse
	err = json.Unmarshal(resp.Body(), &data)
	if err != nil {
		return JmxResponse{}, err
	}
	return data, nil
}

func processNode(ch chan<- prometheus.Metric, node nodeConfig) {

	jmxData, err := fetchJmxData(node.jmxUrl)
	if err != nil {
		fmt.Printf("Error fetching JMX data from %s: %v\n", node.jmxUrl, err)
		return
	}
	metricStruct := convertSliceToMap(jmxData.Beans)
	dataExporter := config.Memory{}
	rawData, ok := metricStruct[metric].(map[string]interface{})
	if !ok {
		fmt.Printf("Metric %s not found in JMX data\n", metric)
	}
	jsonData, err := json.Marshal(rawData)
	if err != nil {
		fmt.Printf("Error marshalling JSON data: %v\n", err)
		return
	}
	err = json.Unmarshal(jsonData, &dataExporter)
	if err != nil {
		fmt.Printf("Error unmarshalling JSON data: %v\n", err)
		return
	}
	duration := dataExporter.LastGcInfo.Duration
	totalCommitedAfterGc := float64(0)
	totalInitAfterGc := float64(0)
	totalUsedAfterGc := float64(0)
	totalMaxAfterGc := float64(0)
	totalCommitedBeforeGc := float64(0)
	totalInitBeforeGc := float64(0)
	totalUsedBeforeGc := float64(0)
	totalMaxBeforeGc := float64(0)

	for _, m := range dataExporter.LastGcInfo.MemoryUsageAfterGc {
		totalCommitedAfterGc += m.Value.Committed
		totalInitAfterGc += m.Value.Init
		totalUsedAfterGc += m.Value.Used
		totalMaxAfterGc += m.Value.Max
	}
	for _, m := range dataExporter.LastGcInfo.MemoryUsageBeforeGc {
		totalCommitedBeforeGc += m.Value.Committed
		totalInitBeforeGc += m.Value.Init
		totalUsedBeforeGc += m.Value.Used
		totalMaxBeforeGc += m.Value.Max
	}
	for _, m := range listMetric {
		switch m {
		case "totalCommitedAfterGc":
			metricProm := newServerMetric(m, "JMX metric "+m, prometheus.GaugeValue)
			ch <- prometheus.MustNewConstMetric(metricProm.Desc, metricProm.Type, totalCommitedAfterGc, node.ip)
		case "totalInitAfterGc":
			metricProm := newServerMetric(m, "JMX metric "+m, prometheus.GaugeValue)
			ch <- prometheus.MustNewConstMetric(metricProm.Desc, metricProm.Type, totalInitAfterGc, node.ip)
		case "totalUsedAfterGc":
			metricProm := newServerMetric(m, "JMX metric "+m, prometheus.GaugeValue)
			ch <- prometheus.MustNewConstMetric(metricProm.Desc, metricProm.Type, totalUsedAfterGc, node.ip)
		case "totalMaxAfterGc":
			metricProm := newServerMetric(m, "JMX metric "+m, prometheus.GaugeValue)
			ch <- prometheus.MustNewConstMetric(metricProm.Desc, metricProm.Type, totalMaxAfterGc, node.ip)
		case "totalCommitedBeforeGc":
			metricProm := newServerMetric(m, "JMX metric "+m, prometheus.GaugeValue)
			ch <- prometheus.MustNewConstMetric(metricProm.Desc, metricProm.Type, totalCommitedBeforeGc, node.ip)
		case "totalInitBeforeGc":
			metricProm := newServerMetric(m, "JMX metric "+m, prometheus.GaugeValue)
			ch <- prometheus.MustNewConstMetric(metricProm.Desc, metricProm.Type, totalInitBeforeGc, node.ip)
		case "totalUsedBeforeGc":
			metricProm := newServerMetric(m, "JMX metric "+m, prometheus.GaugeValue)
			ch <- prometheus.MustNewConstMetric(metricProm.Desc, metricProm.Type, totalUsedBeforeGc, node.ip)
		case "totalMaxBeforeGc":
			metricProm := newServerMetric(m, "JMX metric "+m, prometheus.GaugeValue)
			ch <- prometheus.MustNewConstMetric(metricProm.Desc, metricProm.Type, totalMaxBeforeGc, node.ip)
		case "duration":
			metricProm := newServerMetric(m, "JMX metric "+m, prometheus.GaugeValue)
			ch <- prometheus.MustNewConstMetric(metricProm.Desc, metricProm.Type, duration, node.ip)
		}
	}
}

// Register the collector with Prometheus
func collectionMetrics(ch chan<- prometheus.Metric) {
	numWorkers := os.Getenv("NUM_WORKERS")
	if numWorkers == "" {
		numWorkers = "3"
	}
	// Convert numWorkers to int
	numWorkersInt, err := strconv.Atoi(numWorkers)
	if err != nil {
		fmt.Printf("Error converting NUM_WORKERS to int: %v\n", err)
		return
	}
	// Create a channel to limit the number of concurrent workers
	workerPool := make(chan struct{}, numWorkersInt)
	var wg sync.WaitGroup
	for _, node := range nodeConfigs {
		if node.jmxUrl == "" {
			continue
		}
		wg.Add(1)
		go func(node nodeConfig) {
			defer wg.Done()
			// Acquire a worker
			workerPool <- struct{}{}
			defer func() { <-workerPool }()
			processNode(ch, node)
		}(node)
	}
	wg.Wait()
}

func main() {
	// Initialize node configurations
	impalaPort := os.Getenv("IMPALA_PORT")
	if impalaPort == "" {
		impalaPort = "25000"
	}
	exporterPort := os.Getenv("PORT")
	if exporterPort == "" {
		exporterPort = "9206"
	}
	ip := os.Getenv("NODE_IP")
	ip = "10.110.69.14,10.110.69.15"
	nodeIP := strings.Split(ip, ",")
	for _, ip := range nodeIP {
		nodeConfigs = append(nodeConfigs, nodeConfig{ip: ip, jmxUrl: fmt.Sprintf("http://%s:%s%s", ip, impalaPort, uriJmx)})
	}
	prometheus.MustRegister(&impalaExporter{})
	http.Handle("/metrics", promhttp.Handler())
	fmt.Printf("Starting server on port %s\n", exporterPort)
	err := http.ListenAndServe(":"+exporterPort, nil)
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		return
	}

}
