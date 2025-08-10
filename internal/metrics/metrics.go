package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
)

// EventsCollected counts the number of Kubernetes events the collector has observed.
var EventsCollected = prometheus.NewCounter(prometheus.CounterOpts{
    Namespace: "kabinet",
    Name:      "events_collected_total",
    Help:      "Total number of Kubernetes events collected by the watcher.",
})

// Init registers all metrics with the default Prometheus registry.
// Keeping registration centralized makes adding new metrics straightforward later.
func Init() {
    prometheus.MustRegister(EventsCollected)
}
