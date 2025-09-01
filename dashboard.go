package main

import (
	"flag"

	sdk "github.com/perses/perses/go-sdk"
	"github.com/perses/perses/go-sdk/dashboard"
	"github.com/perses/perses/go-sdk/panel"
	panelgroup "github.com/perses/perses/go-sdk/panel-group"
	listVar "github.com/perses/perses/go-sdk/variable/list-variable"

	promDs "github.com/perses/plugins/prometheus/sdk/go/datasource"
	"github.com/perses/plugins/prometheus/sdk/go/query"
	labelValuesVar "github.com/perses/plugins/prometheus/sdk/go/variable/label-values"
	ts "github.com/perses/plugins/timeserieschart/sdk/go"
)

func main() {
	flag.Parse()
	exec := sdk.NewExec()

	b, err := dashboard.New(
		// IMPORTANT: metadata.name must be DNS-safe (no spaces)
		"demo-app-prometheus",
		dashboard.ProjectName("demo"),

		// Local Prometheus datasource (ref matches what queries use)
		dashboard.AddDatasource("prom-incluster",
			promDs.Prometheus(
				promDs.HTTPProxy("http://kube-prometheus-stack-prometheus.kube-prometheus-stack.svc:9090"),
			),
		),

		// Example variable (optional)
		dashboard.AddVariable("pod",
			listVar.List(
				labelValuesVar.PrometheusLabelValues("pod",
					labelValuesVar.Matchers(`{namespace="o11y-demo"}`),
					labelValuesVar.Datasource("prom-incluster"),
				),
				listVar.DisplayName("Pod"),
			),
		),

		// Panels
		dashboard.AddPanelGroup("Traffic / Latency / Errors / Inflight",
			panelgroup.PanelsPerLine(2),

			panelgroup.AddPanel("Requests/sec by path",
				ts.Chart(),
				panel.AddQuery(query.PromQL(
					`sum by (path) (rate(http_requests_total{namespace="o11y-demo"}[5m]))`,
					query.Datasource("prom-incluster"),
				)),
			),

			panelgroup.AddPanel("Latency p95",
				ts.Chart(),
				panel.AddQuery(query.PromQL(
					`histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket{namespace="o11y-demo"}[5m])) by (le))`,
					query.Datasource("prom-incluster"),
				)),
			),

			panelgroup.AddPanel("Errors/sec (5xx)",
				ts.Chart(),
				panel.AddQuery(query.PromQL(
					`sum by (path) (rate(http_requests_total{namespace="o11y-demo", code=~"5.."}[5m]))`,
					query.Datasource("prom-incluster"),
				)),
			),

			panelgroup.AddPanel("In-flight requests",
				ts.Chart(),
				panel.AddQuery(query.PromQL(
					`sum(inflight_requests{namespace="o11y-demo"})`,
					query.Datasource("prom-incluster"),
				)),
			),
		),
	)

	// Emit the Dashboard JSON to stdout (percli captures this)
	exec.BuildDashboard(b, err)
}

