package metrics

import (
	"log"
	"sync"
)

// MetricData represents a single metric entry.
type MetricData struct {
	BlockTimestamp int64
	LocalTimestamp int64
	Delay          float64
}

// MetricsHandler handles the aggregation and reporting of metrics.
type MetricsHandler struct {
	instanceUID string
	metrics     []MetricData
	mutex       sync.Mutex
}

// Singleton instance of MetricsHandler.
var handlerInstance *MetricsHandler
var once sync.Once

// GetMetricsHandler returns the singleton instance of MetricsHandler.
func GetMetricsHandler(instanceUID string) *MetricsHandler {
	once.Do(func() {
		handlerInstance = &MetricsHandler{
			instanceUID: instanceUID,
			metrics:     []MetricData{},
		}
	})
	return handlerInstance
}

// AddMetric adds a new metric entry to the handler.
func (m *MetricsHandler) AddMetric(blockTimestamp, localTimestamp int64) {
	delay := float64(localTimestamp-blockTimestamp) / 1e3 // Convert milliseconds to seconds
	metric := MetricData{
		BlockTimestamp: blockTimestamp,
		LocalTimestamp: localTimestamp,
		Delay:          delay,
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.metrics = append(m.metrics, metric)
}

// AggregateAndClear aggregates metrics and clears the stored data.
func (m *MetricsHandler) AggregateAndClear() (int, float64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	totalMessages := len(m.metrics)
	if totalMessages == 0 {
		return 0, 0
	}

	var totalDelay float64
	for _, metric := range m.metrics {
		totalDelay += metric.Delay
	}

	avgDelay := totalDelay / float64(totalMessages)
	m.metrics = []MetricData{} // Clear the metrics after aggregation

	return totalMessages, avgDelay
}

// ReportMetrics sends the aggregated metrics to the specified endpoint.
func (m *MetricsHandler) ReportMetrics(endpoint string) {
	totalMessages, avgDelay := m.AggregateAndClear()

	payload := map[string]interface{}{
		"instanceUID":    m.instanceUID,
		"messagesPerMin": totalMessages,
		"avgDelay":       avgDelay,
	}

	log.Printf("Reporting metrics: %+v\n", payload)

	// Send the payload to the endpoint (HTTP POST request).
	err := SendMetrics(endpoint, payload)
	if err != nil {
		log.Printf("Failed to report metrics: %v\n", err)
	}
}
