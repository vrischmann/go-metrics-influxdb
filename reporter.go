package metflux

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"runtime"
	"sync"
	"time"

	"github.com/influxdata/influxdb/client"
	"github.com/rcrowley/go-metrics"
)

type reporter struct {
	// Reporter Configurations.
	config *Config

	// InfluxXDB Client used to communicate with influxDB.
	client *client.Client

	// Mutex Lock to call the run() only once for multiple Run() called
	// with same registry and configs.
	once sync.Once
}

func New(conf *Config) (*reporter, error) {
	if conf.InfluxDB == nil {
		return nil, errors.New("no influxdb configuration found")
	}

	c, err := conf.InfluxDB.newClient()
	if err != nil {
		return nil, err
	}

	if conf.Interval == time.Duration(0) {
		conf.Interval = time.Second * 10
	}

	if conf.PingInterval == time.Duration(0) {
		conf.PingInterval = time.Second * 5
	}

	return &reporter{
		config: conf,
		client: c,
	}, nil
}

// Creates a New InfluxDB client based on the `InfluDB` configs
func (i *InfluxDB) newClient() (*client.Client, error) {
	u, err := url.Parse(i.URL)
	if err != nil {
		log.Printf("unable to parse InfluxDB url %s. err=%v", i.URL, err)
		return nil, err
	}

	return client.NewClient(client.Config{
		URL:      *u,
		Username: i.Username,
		Password: i.Password,
	})
}

func (r *reporter) Run() {
	r.once.Do(func() {
		go func() {
			defer func() {
				if rec := recover(); rec != nil {
					r.handlePanic(rec)
				}
			}()
			r.run()
		}()
	})
}

func (r *reporter) run() {
	intervalTicker := time.Tick(r.config.Interval)
	pingTicker := time.Tick(r.config.PingInterval)

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
				if r.client, err = r.config.InfluxDB.newClient(); err != nil {
					log.Printf("unable to make InfluxDB client. err=%v", err)
				}
			}
		}
	}
}

func (r *reporter) send() error {
	var pts []client.Point
	r.config.Registry.Each(func(name string, i interface{}) {
		now := time.Now()
		matrixFound := false

		fmt.Println("inside", name, i)
		switch m := i.(type) {
		case metrics.Counter:
			pts = append(pts, client.Point{
				Measurement: fmt.Sprintf("%s.count", name),
				Tags:        r.config.Tags,
				Fields: map[string]interface{}{
					"value": m.Count(),
				},
				Time: now,
			})
			matrixFound = true
		case metrics.Gauge:
			pts = append(pts, client.Point{
				Measurement: fmt.Sprintf("%s.gauge", name),
				Tags:        r.config.Tags,
				Fields: map[string]interface{}{
					"value": m.Value(),
				},
				Time: now,
			})
			matrixFound = true
		case metrics.GaugeFloat64:
			pts = append(pts, client.Point{
				Measurement: fmt.Sprintf("%s.gauge", name),
				Tags:        r.config.Tags,
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
				Tags:        r.config.Tags,
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
				Tags:        r.config.Tags,
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
				Tags:        r.config.Tags,
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
		default:
			for _, customMetric := range r.config.CustomMetrics {
				if customMetric.IsApplicable(i) {
					pts = append(pts, customMetric.GetPointFunction(i))
					matrixFound = true
					break
				}
			}
		}
		if matrixFound {
			for _, callback := range r.config.Callbacks {
				callback(i)
			}
		}
	})

	bps := client.BatchPoints{
		Points:   pts,
		Database: r.config.Database,
	}
	_, err := r.client.Write(bps)
	return err
}

func (r *reporter) handlePanic(rec interface{}) {
	logPanic(rec)

	// Additional panic handlers to run
	for _, f := range r.config.PanicHandlers {
		f(r)
	}
}

func logPanic(r interface{}) {
	callers := ""
	for i := 2; true; i++ {
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		callers = callers + fmt.Sprintf("%v:%v\n", file, line)
	}
	log.Printf("Recovered from panic: %#v \n%v", r, callers)
}
