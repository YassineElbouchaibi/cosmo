package integration

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/wundergraph/cosmo/router-tests/testenv"
	"github.com/wundergraph/cosmo/router/pkg/otel"
	"github.com/wundergraph/cosmo/router/pkg/trace/tracetest"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"testing"
)

func TestTelemetry(t *testing.T) {
	t.Parallel()

	const employeesIDData = `{"data":{"employees":[{"id":1},{"id":2},{"id":3},{"id":4},{"id":5},{"id":7},{"id":8},{"id":10},{"id":11},{"id":12}]}}`

	t.Run("Trace unnamed GraphQL operation with metrics", func(t *testing.T) {
		t.Parallel()

		metricReader := metric.NewManualReader()
		exporter := tracetest.NewInMemoryExporter(t)

		testenv.Run(t, &testenv.Config{
			TraceExporter: exporter,
			MetricReader:  metricReader,
		}, func(t *testing.T, xEnv *testenv.Environment) {
			res := xEnv.MakeGraphQLRequestOK(testenv.GraphQLRequest{
				Query: `query { employees { id } }`,
			})
			require.JSONEq(t, employeesIDData, res.Body)

			sn := exporter.GetSpans().Snapshots()
			require.Len(t, sn, 8, "expected 8 spans, got %d", len(sn))

			/**
			* Spans
			 */

			// Pre-Handler Operation steps
			require.Equal(t, "Operation - Parse", sn[0].Name())
			require.Equal(t, trace.SpanKindInternal, sn[0].SpanKind())
			require.Equal(t, sdktrace.Status{Code: codes.Unset}, sn[0].Status())

			require.Equal(t, "Operation - Normalize", sn[1].Name())
			require.Equal(t, trace.SpanKindInternal, sn[1].SpanKind())
			require.Equal(t, sdktrace.Status{Code: codes.Unset}, sn[1].Status())

			require.Equal(t, "Operation - Validate", sn[2].Name())
			require.Equal(t, trace.SpanKindInternal, sn[2].SpanKind())
			require.Equal(t, sdktrace.Status{Code: codes.Unset}, sn[2].Status())

			require.Equal(t, "Operation - Plan", sn[3].Name())
			require.Equal(t, trace.SpanKindInternal, sn[3].SpanKind())
			require.Equal(t, sdktrace.Status{Code: codes.Unset}, sn[3].Status())

			// Engine Transport
			require.Equal(t, "query unnamed", sn[4].Name())
			require.Equal(t, trace.SpanKindClient, sn[4].SpanKind())
			require.Equal(t, sdktrace.Status{Code: codes.Unset}, sn[4].Status())

			// Engine Loader Hooks
			require.Equal(t, "Engine - Fetch", sn[5].Name())
			require.Equal(t, trace.SpanKindInternal, sn[5].SpanKind())
			require.Equal(t, sdktrace.Status{Code: codes.Unset}, sn[5].Status())

			// GraphQL handler
			require.Equal(t, "Operation - Execute", sn[6].Name())
			require.Equal(t, trace.SpanKindInternal, sn[6].SpanKind())
			require.Equal(t, sdktrace.Status{Code: codes.Unset}, sn[6].Status())

			// Root Server middleware
			require.Equal(t, "query unnamed", sn[7].Name())
			require.Equal(t, trace.SpanKindServer, sn[7].SpanKind())
			require.Equal(t, sdktrace.Status{Code: codes.Unset}, sn[7].Status())

			/**
			* Metrics
			 */
			rm := metricdata.ResourceMetrics{}
			err := metricReader.Collect(context.Background(), &rm)
			require.NoError(t, err)

			httpRequestsMetric := metricdata.Metrics{
				Name:        "router.http.requests",
				Description: "Total number of requests",
				Unit:        "",
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: true,
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								otel.WgClientName.String("unknown"),
								otel.WgClientVersion.String("missing"),
								otel.WgFederatedGraphID.String("graph"),
								otel.WgOperationHash.String("14226210703439426856"),
								otel.WgOperationName.String(""),
								otel.WgOperationProtocol.String("http"),
								otel.WgOperationType.String("query"),
								otel.WgRouterClusterName.String(""),
								otel.WgRouterConfigVersion.String(""),
								otel.WgRouterVersion.String("dev"),
								otel.WgSubgraphID.String("0"),
								otel.WgSubgraphName.String("employees"),
							),
							Value: 1,
						},
						{
							Attributes: attribute.NewSet(
								semconv.HTTPStatusCode(200),
								otel.WgClientName.String("unknown"),
								otel.WgClientVersion.String("missing"),
								otel.WgFederatedGraphID.String("graph"),
								otel.WgOperationHash.String("14226210703439426856"),
								otel.WgOperationName.String(""),
								otel.WgOperationProtocol.String("http"),
								otel.WgOperationType.String("query"),
								otel.WgRouterClusterName.String(""),
								otel.WgRouterConfigVersion.String(""),
								otel.WgRouterVersion.String("dev"),
							),
							Value: 1,
						},
					},
				},
			}

			requestDurationMetric := metricdata.Metrics{
				Name:        "router.http.request.duration_milliseconds",
				Description: "Server latency in milliseconds",
				Unit:        "ms",
				Data: metricdata.Histogram[float64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[float64]{
						{
							Attributes: attribute.NewSet(
								otel.WgClientName.String("unknown"),
								otel.WgClientVersion.String("missing"),
								otel.WgFederatedGraphID.String("graph"),
								otel.WgOperationHash.String("14226210703439426856"),
								otel.WgOperationName.String(""),
								otel.WgOperationProtocol.String("http"),
								otel.WgOperationType.String("query"),
								otel.WgRouterClusterName.String(""),
								otel.WgRouterConfigVersion.String(""),
								otel.WgRouterVersion.String("dev"),
								otel.WgSubgraphID.String("0"),
								otel.WgSubgraphName.String("employees"),
							),
							Sum: 0,
						},
						{
							Attributes: attribute.NewSet(
								semconv.HTTPStatusCode(200),
								otel.WgClientName.String("unknown"),
								otel.WgClientVersion.String("missing"),
								otel.WgFederatedGraphID.String("graph"),
								otel.WgOperationHash.String("14226210703439426856"),
								otel.WgOperationName.String(""),
								otel.WgOperationProtocol.String("http"),
								otel.WgOperationType.String("query"),
								otel.WgRouterClusterName.String(""),
								otel.WgRouterConfigVersion.String(""),
								otel.WgRouterVersion.String("dev"),
							),
							Sum: 0,
						},
					},
				},
			}

			requestContentLengthMetric := metricdata.Metrics{
				Name:        "router.http.request.content_length",
				Description: "Total number of request bytes",
				Unit:        "bytes",
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: true,
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								otel.WgClientName.String("unknown"),
								otel.WgClientVersion.String("missing"),
								otel.WgFederatedGraphID.String("graph"),
								otel.WgOperationHash.String("14226210703439426856"),
								otel.WgOperationName.String(""),
								otel.WgOperationProtocol.String("http"),
								otel.WgOperationType.String("query"),
								otel.WgRouterClusterName.String(""),
								otel.WgRouterConfigVersion.String(""),
								otel.WgRouterVersion.String("dev"),
								otel.WgSubgraphID.String("0"),
								otel.WgSubgraphName.String("employees"),
							),
							Value: 28,
						},
						{
							Attributes: attribute.NewSet(
								semconv.HTTPStatusCode(200),
								otel.WgClientName.String("unknown"),
								otel.WgClientVersion.String("missing"),
								otel.WgFederatedGraphID.String("graph"),
								otel.WgOperationHash.String("14226210703439426856"),
								otel.WgOperationName.String(""),
								otel.WgOperationProtocol.String("http"),
								otel.WgOperationType.String("query"),
								otel.WgRouterClusterName.String(""),
								otel.WgRouterConfigVersion.String(""),
								otel.WgRouterVersion.String("dev"),
							),
							Value: 38,
						},
					},
				},
			}

			responseContentLengthMetric := metricdata.Metrics{
				Name:        "router.http.response.content_length",
				Description: "Total number of response bytes",
				Unit:        "bytes",
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					IsMonotonic: true,
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								semconv.HTTPStatusCode(200),
								otel.WgClientName.String("unknown"),
								otel.WgClientVersion.String("missing"),
								otel.WgFederatedGraphID.String("graph"),
								otel.WgOperationHash.String("14226210703439426856"),
								otel.WgOperationName.String(""),
								otel.WgOperationProtocol.String("http"),
								otel.WgOperationType.String("query"),
								otel.WgRouterClusterName.String(""),
								otel.WgRouterConfigVersion.String(""),
								otel.WgRouterVersion.String("dev"),
								otel.WgSubgraphID.String("0"),
								otel.WgSubgraphName.String("employees"),
							),
							Value: 117,
						},
						{
							Attributes: attribute.NewSet(
								semconv.HTTPStatusCode(200),
								otel.WgClientName.String("unknown"),
								otel.WgClientVersion.String("missing"),
								otel.WgFederatedGraphID.String("graph"),
								otel.WgOperationHash.String("14226210703439426856"),
								otel.WgOperationName.String(""),
								otel.WgOperationProtocol.String("http"),
								otel.WgOperationType.String("query"),
								otel.WgRouterClusterName.String(""),
								otel.WgRouterConfigVersion.String(""),
								otel.WgRouterVersion.String("dev"),
							),
							Value: 117,
						},
					},
				},
			}

			requestInFlightMetric := metricdata.Metrics{
				Name:        "router.http.requests.in_flight.count",
				Description: "Number of requests in flight",
				Unit:        "",
				Data: metricdata.Sum[int64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.DataPoint[int64]{
						{
							Attributes: attribute.NewSet(
								otel.WgFederatedGraphID.String("graph"),
								otel.WgRouterClusterName.String(""),
								otel.WgRouterConfigVersion.String(""),
								otel.WgRouterVersion.String("dev"),
							),
							Value: 0,
						},
						{
							Attributes: attribute.NewSet(
								otel.WgClientName.String("unknown"),
								otel.WgClientVersion.String("missing"),
								otel.WgFederatedGraphID.String("graph"),
								otel.WgOperationHash.String("14226210703439426856"),
								otel.WgOperationName.String(""),
								otel.WgOperationProtocol.String("http"),
								otel.WgOperationType.String("query"),
								otel.WgRouterClusterName.String(""),
								otel.WgRouterConfigVersion.String(""),
								otel.WgRouterVersion.String("dev"),
								otel.WgSubgraphID.String("0"),
								otel.WgSubgraphName.String("employees"),
							),
							Value: 0,
						},
					},
				},
			}

			want := metricdata.ScopeMetrics{
				Scope: instrumentation.Scope{
					Name:      "cosmo.router",
					SchemaURL: "",
					Version:   "0.0.1",
				},
				Metrics: []metricdata.Metrics{
					httpRequestsMetric,
					requestDurationMetric,
					requestContentLengthMetric,
					responseContentLengthMetric,
					requestInFlightMetric,
				},
			}

			require.Equal(t, 1, len(rm.ScopeMetrics), "expected 1 ScopeMetrics, got %d", len(rm.ScopeMetrics))
			require.Equal(t, 5, len(rm.ScopeMetrics[0].Metrics), "expected 5 Metrics, got %d", len(rm.ScopeMetrics[0].Metrics))

			metricdatatest.AssertEqual(t, want, rm.ScopeMetrics[0], metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())

			metricdatatest.AssertEqual(t, httpRequestsMetric, rm.ScopeMetrics[0].Metrics[0], metricdatatest.IgnoreTimestamp())
			metricdatatest.AssertEqual(t, requestContentLengthMetric, rm.ScopeMetrics[0].Metrics[2], metricdatatest.IgnoreTimestamp())
			metricdatatest.AssertEqual(t, responseContentLengthMetric, rm.ScopeMetrics[0].Metrics[3], metricdatatest.IgnoreTimestamp())
			metricdatatest.AssertEqual(t, requestInFlightMetric, rm.ScopeMetrics[0].Metrics[4], metricdatatest.IgnoreTimestamp())

		})
	})

	t.Run("Trace named operation", func(t *testing.T) {
		t.Parallel()

		exporter := tracetest.NewInMemoryExporter(t)

		testenv.Run(t, &testenv.Config{
			TraceExporter: exporter,
		}, func(t *testing.T, xEnv *testenv.Environment) {
			res := xEnv.MakeGraphQLRequestOK(testenv.GraphQLRequest{
				Query: `query myQuery { employees { id } }`,
			})
			require.JSONEq(t, employeesIDData, res.Body)

			sn := exporter.GetSpans().Snapshots()
			require.Len(t, sn, 8, "expected 8 spans, got %d", len(sn))

			// Pre-Handler Operation steps
			require.Equal(t, "Operation - Parse", sn[0].Name())
			require.Equal(t, trace.SpanKindInternal, sn[0].SpanKind())
			require.Equal(t, sdktrace.Status{Code: codes.Unset}, sn[0].Status())

			require.Equal(t, "Operation - Normalize", sn[1].Name())
			require.Equal(t, trace.SpanKindInternal, sn[1].SpanKind())
			require.Equal(t, sdktrace.Status{Code: codes.Unset}, sn[1].Status())

			require.Equal(t, "Operation - Validate", sn[2].Name())
			require.Equal(t, trace.SpanKindInternal, sn[2].SpanKind())
			require.Equal(t, sdktrace.Status{Code: codes.Unset}, sn[2].Status())

			require.Equal(t, "Operation - Plan", sn[3].Name())
			require.Equal(t, trace.SpanKindInternal, sn[3].SpanKind())
			require.Equal(t, sdktrace.Status{Code: codes.Unset}, sn[3].Status())

			// Engine Transport
			require.Equal(t, "query myQuery", sn[4].Name())
			require.Equal(t, trace.SpanKindClient, sn[4].SpanKind())
			require.Equal(t, sdktrace.Status{Code: codes.Unset}, sn[4].Status())

			// Engine Loader Hooks
			require.Equal(t, "Engine - Fetch", sn[5].Name())
			require.Equal(t, trace.SpanKindInternal, sn[5].SpanKind())
			require.Equal(t, sdktrace.Status{Code: codes.Unset}, sn[5].Status())

			// GraphQL handler
			require.Equal(t, "Operation - Execute", sn[6].Name())
			require.Equal(t, trace.SpanKindInternal, sn[6].SpanKind())
			require.Equal(t, sdktrace.Status{Code: codes.Unset}, sn[6].Status())

			// Root Server middleware
			require.Equal(t, "query myQuery", sn[7].Name())
			require.Equal(t, trace.SpanKindServer, sn[7].SpanKind())
			require.Equal(t, sdktrace.Status{Code: codes.Unset}, sn[7].Status())
		})
	})
}
