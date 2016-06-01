package metflux

import (
	"time"

	"github.com/influxdata/influxdb/client"
	"github.com/rcrowley/go-metrics"
)

type Config struct {
	*InfluxDB

	// Registry holds the metricses. That were needed to be
	// reported to influxDB.
	Registry metrics.Registry

	// Time inerval between two consicutive influxDB call to
	// store the matrix value to the DB. If not set Default Will
	// be 10secs.
	Interval time.Duration

	// Time interval to make sure the connection betwwen influxDB
	// and client are alive. if the ping failed with an error client
	// will try to reconnect to influxDB, if thus failed will throw
	// panics.
	PingInterval time.Duration

	// List of callback functions that will be invoked after every influxDB
	// call, with the metrics interface that was used to read as param.
	Callbacks []Callback

	// List of custom matrices, that will be matched against the registry
	// and needs to read metric data from. If any method from the list is
	// invoked this stops matching and set as metrics Read.
	CustomMetrics []CustomMetric

	// PanicHandlers are the handlers to call whenever a panic occers.
	PanicHandlers []func(interface{})
}

type InfluxDB struct {
	// InfluxDB url to connect.
	URL      string
	Database string
	Username string
	Password string

	// Custom InfluxDB Tags those will be send to influxDB with every call.
	Tags map[string]string
}

// Callbacks invoked after every metric read. the parameter is the metric
// that was read it self.
type Callback func(interface{})

// CustomMetrics type added support to send custom metric data to be sent
// to InfluxDB. This calls IsApplicable with the metric as parameter to match the type
// If IsApplicable returns true it calls GetPointFunction to get the influx point from the metric
type CustomMetric struct {
	// Reflect.Type of underlying metric interface{} Type. That will compared
	// to the metric read.
	IsApplicable func(i interface{}) bool

	// GetPointFunction will invoke when the metric type matched this metricType.
	// This will expect only one parameter the underlying metric type and return
	// a influxDB point parsed from that metric/
	GetPointFunction GetPointFuncs
}

// Applicability function. Receives the metric as a parameter and returns true
// If the metric is matched to this CustomMetric.
type Applicable func(interface{}) bool

// This will expect only one parameter the underlying metric type and return
// a influxDB point parsed from that metric/
type GetPointFuncs func(interface{}) client.Point

func NewCustomMetric(a Applicable, getPointFunction GetPointFuncs) CustomMetric {
	return CustomMetric{
		IsApplicable:     a,
		GetPointFunction: getPointFunction,
	}
}
