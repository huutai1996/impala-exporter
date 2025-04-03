package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

const (
	uriJmx       = "/jmx"
	exporterPort = "9206"
	impalaPort   = "25000"
)

var (
	nodeIP      = []string{"10.110.69.15"}
	nodeConfigs []nodeConfig
)

var metricMapGauges = map[string]string{
	"java.lang:type=MemoryPool,name=Metaspace":  "jvm_memory_pool_metaspace_usage_used",
	"java.lang:type=MemoryPool,name=PS Old Gen": "jvm_memory_pool_ps_old_gen_usage_used",
}

type JmxResponse struct {
	Beans []map[string]interface{} `json:"beans"`
}

type nodeConfig struct {
	ip     string
	jmxUrl string
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

// Register the collector with Prometheus
func collectionMetrics(registry *prometheus.Registry) {
	for _, node := range nodeConfigs {
		if node.jmxUrl == "" {
			continue
		}
		jmxData, err := fetchJmxData(node.jmxUrl)
		if err != nil {
			fmt.Printf("Error fetching JMX data from %s: %v\n", node.jmxUrl, err)
			continue
		}
		metricStruct := convertSliceToMap(jmxData.Beans)
		for k, v := range metricMapGauges {
			metricData, err := json.Marshal(metricStruct[k])
			if err != nil {
				fmt.Println("Error marshalling metric data")
			}

			gauge := prometheus.NewGauge(prometheus.GaugeOpts{
				Name:        v,
				ConstLabels: prometheus.Labels{"Type": metricData.Type},
			})
			registry.MustRegister(gauge)
			gauge.Set(float64(metricData.Usage.Used))
		}
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

func main() {
	// Initialize node configurations
	for _, ip := range nodeIP {
		nodeConfigs = append(nodeConfigs, nodeConfig{ip: ip, jmxUrl: fmt.Sprintf("http://%s:%s%s", ip, impalaPort, uriJmx)})
	}
	registry := prometheus.NewRegistry()
	collectionMetrics(registry)
	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	fmt.Printf("Starting exporter on port %s\n", exporterPort)
	http.ListenAndServe(":"+exporterPort, nil)
}
