package influxdb

import (
	"fmt"
	"log"
	"net/url"
	"sync"
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
	
	// List of custom metricses, that will be matched against the registry
	// and needs to read metric data from.
	CustomMetrics []CustomMetrics
	
	// PanicHandlers are the handlers to call whenever a panic occers.
	PanicHandlers []func(interface{})
}

type InfluxDB struct {
	// InfluxDB url to connect.
	URL string
	Database string
	Username string
	Password string
	
	// Custom InfluxDB Tags those will be send to influxDB with every call.
	Tags map[string]string
}

type reporter struct {
	// Reporter Configurations.
	config *Config
	
	// InfluXDB Client used to communicate with influxDB.
	client *client.Client
	
	// Mutex Lock to call the run() only once for multiple Run() calles
	// with same registry and configs.
	once sync.Once
	
	mu sync.Mutex
}

type Callback func(interface{})

type CustomMetrics interface{}

func New(conf *Config) (*reporter, error) {
	if conf.InfluxDB == nil {
		return nil, errors.New("no influxdb configuration found")
	}
	
	c, err := conf.InfluxDB.newClient()
	if err != nil {
		return nil, err
	}
	
	if conf.Interval == time.Duration(0) {
		conf.Interval = time.Seconds*10
	}
	
	if conf.PingInterval == time.Duration(0) {
		conf.PingInterval = time.Seconds*5
	}
	
	return &reporter{
		config: conf,
		client: c,
	}, nil
}

// Creates a New InfluxDB client based on the `InfluDB` configs
func (i *InfluxDB) newClient() (*client.Client, error) {
	u, err := url.Parse(url)
	if err != nil {
		log.Printf("unable to parse InfluxDB url %s. err=%v", url, err)
		return nil, err
	}

	return client.NewClient(client.Config{
		URL:      u,
		Username: r.username,
		Password: r.password,
	})
}

func (r *reporter) Run() {
	r.once.Do(func(){
		go func() {
			defer func() {
				if r := recover(); r != nil {
					r.handlePanic()
				}
			}()
			r.run()
		}()
	})
}

func (r *reporter) run() {
	intervalTicker := time.Tick(r.Interval)
	pingTicker := time.Tick(r.PingInterval)

	for {
		select {
		case <-intervalTicker:
			if err := r.send(); err != nil {
				log.Printf("unable to send metrics to InfluxDB. err=%v", err)
			}
		case <-pingTicker:
			_, _, err := r.client.Ping()
			if err != nil {
				log.Printf("got error while sending a ping to InfluxDB, trying to recreate client. err=%v", err)

				if err = r.makeClient(); err != nil {
					log.Printf("unable to make InfluxDB client. err=%v", err)
				}
			}
		}
	}
}

func (r *reporter) send() error {
	var pts []client.Point

	r.reg.Each(func(name string, i interface{}) {
		now := time.Now()
		matrixFound := false

		switch m := i.(type) {
		case metrics.Counter:
			pts = append(pts, client.Point{
				Measurement: fmt.Sprintf("%s.count", name),
				Tags:        r.tags,
				Fields: map[string]interface{}{
					"value": m.Count(),
				},
				Time: now,
			})
			matrixFound = true
		case metrics.Gauge:
			pts = append(pts, client.Point{
				Measurement: fmt.Sprintf("%s.gauge", name),
				Tags:        r.tags,
				Fields: map[string]interface{}{
					"value": m.Value(),
				},
				Time: now,
			})
			matrixFound = true
		case metrics.GaugeFloat64:
			pts = append(pts, client.Point{
				Measurement: fmt.Sprintf("%s.gauge", name),
				Tags:        r.tags,
				Fields: map[string]interface{}{
					"value": m.Value(),
				},
				Time: now,
			})
			matrixFound = true
		case metrics.Histogram:
			ps := m.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999, 0.9999})
			pts = append(pts, client.Point{
				Measurement: fmt.Sprintf("%s.histogram", name),
				Tags:        r.tags,
				Fields: map[string]interface{}{
					"count":    m.Count(),
					"max":      m.Max(),
					"mean":     m.Mean(),
					"min":      m.Min(),
					"stddev":   m.StdDev(),
					"variance": m.Variance(),
					"p50":      ps[0],
					"p75":      ps[1],
					"p95":      ps[2],
					"p99":      ps[3],
					"p999":     ps[4],
					"p9999":    ps[5],
				},
				Time: now,
			})
			matrixFound = true
		case metrics.Meter:
			pts = append(pts, client.Point{
				Measurement: fmt.Sprintf("%s.meter", name),
				Tags:        r.tags,
				Fields: map[string]interface{}{
					"count": m.Count(),
					"m1":    m.Rate1(),
					"m5":    m.Rate5(),
					"m15":   m.Rate15(),
					"mean":  m.RateMean(),
				},
				Time: now,
			})
			matrixFound = true
		case metrics.Timer:
			ps := m.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999, 0.9999})
			pts = append(pts, client.Point{
				Measurement: fmt.Sprintf("%s.timer", name),
				Tags:        r.tags,
				Fields: map[string]interface{}{
					"count":    m.Count(),
					"max":      m.Max(),
					"mean":     m.Mean(),
					"min":      m.Min(),
					"stddev":   m.StdDev(),
					"variance": m.Variance(),
					"p50":      ps[0],
					"p75":      ps[1],
					"p95":      ps[2],
					"p99":      ps[3],
					"p999":     ps[4],
					"p9999":    ps[5],
					"m1":       m.Rate1(),
					"m5":       m.Rate5(),
					"m15":      m.Rate15(),
					"meanrate": m.RateMean(),
				},
				Time: now,
			})
			matrixFound = true
		}
		if r.callback != nil && matrixFound {
			r.callback(i)
		}
	})

	bps := client.BatchPoints{
		Points:   pts,
		Database: r.database,
	}

	_, err := r.client.Write(bps)
	return err
}
