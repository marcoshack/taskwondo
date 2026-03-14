package metrics

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
)

// DBCollector collects database connection pool metrics from sql.DB.Stats().
type DBCollector struct {
	db *sql.DB

	openDesc     *prometheus.Desc
	idleDesc     *prometheus.Desc
	inUseDesc    *prometheus.Desc
	waitDesc     *prometheus.Desc
	waitDurDesc  *prometheus.Desc
}

// NewDBCollector creates a collector that reads sql.DB.Stats() on each scrape.
func NewDBCollector(db *sql.DB) *DBCollector {
	return &DBCollector{
		db: db,
		openDesc: prometheus.NewDesc(
			namespace+"_db_connections_open",
			"Number of open connections to the database.",
			nil, nil,
		),
		idleDesc: prometheus.NewDesc(
			namespace+"_db_connections_idle",
			"Number of idle connections in the pool.",
			nil, nil,
		),
		inUseDesc: prometheus.NewDesc(
			namespace+"_db_connections_in_use",
			"Number of connections currently in use.",
			nil, nil,
		),
		waitDesc: prometheus.NewDesc(
			namespace+"_db_connections_wait_total",
			"Total number of connections waited for.",
			nil, nil,
		),
		waitDurDesc: prometheus.NewDesc(
			namespace+"_db_connections_wait_duration_seconds_total",
			"Total time blocked waiting for a new connection.",
			nil, nil,
		),
	}
}

// Describe sends the descriptor for each metric.
func (c *DBCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.openDesc
	ch <- c.idleDesc
	ch <- c.inUseDesc
	ch <- c.waitDesc
	ch <- c.waitDurDesc
}

// Collect reads sql.DB.Stats() and sends the current values.
func (c *DBCollector) Collect(ch chan<- prometheus.Metric) {
	stats := c.db.Stats()

	ch <- prometheus.MustNewConstMetric(c.openDesc, prometheus.GaugeValue, float64(stats.OpenConnections))
	ch <- prometheus.MustNewConstMetric(c.idleDesc, prometheus.GaugeValue, float64(stats.Idle))
	ch <- prometheus.MustNewConstMetric(c.inUseDesc, prometheus.GaugeValue, float64(stats.InUse))
	ch <- prometheus.MustNewConstMetric(c.waitDesc, prometheus.CounterValue, float64(stats.WaitCount))
	ch <- prometheus.MustNewConstMetric(c.waitDurDesc, prometheus.CounterValue, stats.WaitDuration.Seconds())
}
