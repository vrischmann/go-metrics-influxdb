package metflux

import (
	"time"

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
