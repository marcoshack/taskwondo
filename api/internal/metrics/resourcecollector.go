package metrics

import (
	"context"
	"database/sql"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// ResourceCollector collects resource count metrics from the database.
// On each Prometheus scrape it runs COUNT queries (excluding soft-deleted rows).
type ResourceCollector struct {
	db *sql.DB

	usersDesc      *prometheus.Desc
	namespacesDesc *prometheus.Desc
	projectsDesc   *prometheus.Desc
	workItemsDesc  *prometheus.Desc
	milestonesDesc *prometheus.Desc
}

// NewResourceCollector creates a collector that counts database resources on each scrape.
func NewResourceCollector(db *sql.DB) *ResourceCollector {
	return &ResourceCollector{
		db: db,
		usersDesc: prometheus.NewDesc(
			namespace+"_users_total",
			"Current number of users.",
			nil, nil,
		),
		namespacesDesc: prometheus.NewDesc(
			namespace+"_namespaces_total",
			"Current number of namespaces.",
			nil, nil,
		),
		projectsDesc: prometheus.NewDesc(
			namespace+"_projects_total",
			"Current number of projects.",
			nil, nil,
		),
		workItemsDesc: prometheus.NewDesc(
			namespace+"_work_items_total",
			"Current number of work items by status category and type.",
			[]string{"status_category", "type"}, nil,
		),
		milestonesDesc: prometheus.NewDesc(
			namespace+"_milestones_total",
			"Current number of milestones.",
			nil, nil,
		),
	}
}

// Describe sends the descriptor for each metric.
func (c *ResourceCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.usersDesc
	ch <- c.namespacesDesc
	ch <- c.projectsDesc
	ch <- c.workItemsDesc
	ch <- c.milestonesDesc
}

// Collect runs COUNT queries and sends the current values.
func (c *ResourceCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c.collectSimpleCount(ctx, ch, "SELECT COUNT(*) FROM users", c.usersDesc)
	c.collectSimpleCount(ctx, ch, "SELECT COUNT(*) FROM namespaces", c.namespacesDesc)
	c.collectSimpleCount(ctx, ch, "SELECT COUNT(*) FROM projects WHERE deleted_at IS NULL", c.projectsDesc)
	c.collectSimpleCount(ctx, ch, "SELECT COUNT(*) FROM milestones", c.milestonesDesc)
	c.collectWorkItems(ctx, ch)
}

func (c *ResourceCollector) collectSimpleCount(ctx context.Context, ch chan<- prometheus.Metric, query string, desc *prometheus.Desc) {
	var count int64
	if err := c.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		log.Ctx(ctx).Error().Err(err).Str("query", query).Msg("metrics: failed to count resources")
		return
	}
	ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, float64(count))
}

func (c *ResourceCollector) collectWorkItems(ctx context.Context, ch chan<- prometheus.Metric) {
	const query = `
		SELECT ws.category, wi.type, COUNT(*)
		FROM work_items wi
		JOIN projects p ON p.id = wi.project_id
		LEFT JOIN project_type_workflows ptw
			ON ptw.project_id = p.id AND ptw.work_item_type = wi.type
		JOIN workflow_statuses ws
			ON ws.workflow_id = COALESCE(ptw.workflow_id, p.default_workflow_id)
			AND ws.name = wi.status
		WHERE wi.deleted_at IS NULL AND p.deleted_at IS NULL
		GROUP BY ws.category, wi.type`

	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("metrics: failed to count work items")
		return
	}
	defer rows.Close()

	for rows.Next() {
		var category, itemType string
		var count int64
		if err := rows.Scan(&category, &itemType, &count); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("metrics: failed to scan work item count row")
			return
		}
		ch <- prometheus.MustNewConstMetric(c.workItemsDesc, prometheus.GaugeValue, float64(count), category, itemType)
	}
	if err := rows.Err(); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("metrics: work item count rows iteration error")
	}
}
