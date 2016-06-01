# metflux
A influxDB reporter for the [go-metrics](https://github.com/rcrowley/go-metrics) library,
which will post the metrics to [InfluxDB](https://influxdb.com/).


## Origin
This library is a fork of [go-metrics-influxdb](https://github.com/vrischmann/go-metrics-influxdb)
Thanx @vrischmann to write up the library.


## Note
This is only compatible with InfluxDB 0.9+.

**Get this package using**
```bash
$ go get github.com/sadlil/metflux
```

## Usage
```go
import (
  "github.com/sadlil/metflax"
  "github.com/rcrowley/go-metrics"
)

config := &metflux.Config{
  InfluxDB: &metflux.InfluxDB{
  			URL:      "url",
  			Database: "dbname",
  			Username: "username",
  			Password: "password",
  		},
  		Registry: metrics.NewRegistry(),
  		Interval: time.Second * 5,  
}

reporter, err := metflux.New(config) // Creates a new influxDB Reporter
if err == nil {
  // Run the reporter as follows.
  // You do not need to run this into a go routine.
  // run calls its self inside an go routine. In cause of any
  // panics it will not crash the program, it will log the panic
  // and runs any panic Handler you supplied in PanicHandlers in config.
  // You need not worry about calling Run multiple times, calling run 
  // multiple time do not create InfluxDB client multiple times or neither 
  // it creates two separate reporter. It will create only client and one 
  // watcher, per metflux.Reporter Instance, doesn't matterhow many times 
  // you call it. 
	reporter.Run()
	
	// Run will start the reporter that will watch the registry and report 
	// to influxDB after every Interal you specified.
}
```

#### Other Config Options
```go
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

```

## License
Licensed under the MIT license. See the [LICENSE](LICENSE) file for details.
