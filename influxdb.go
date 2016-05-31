package influxdb

import (
	"fmt"
	"log"
	uurl "net/url"
	"time"

	"github.com/influxdata/influxdb/client"
	"github.com/rcrowley/go-metrics"
)

type reporter struct {
	reg      metrics.Registry
	interval time.Duration

	// if callback is set this will called after every read operation
	// to matrix registry from inside the go-routine with the matrix
	// interface type as the parameters.
	callback Callback

	url      uurl.URL
	database string
	username string
	password string
	tags     map[string]string

	client *client.Client
}

type Callback func(interface{})

// InfluxDB starts a InfluxDB reporter which will post the metrics from the given registry at each d interval.
// if callback is set then it call after reading the registry with the matrix interface.
func InfluxDB(r metrics.Registry, d time.Duration, callback Callback, url, database, username, password string) {
	InfluxDBWithTags(r, d, callback, url, database, username, password, nil)
}

// InfluxDB starts a InfluxDB reporter which will post the metrics from the given registry at each d interval with the specified tags
// if callback is set then it call after reading the registry with the matrix interface.
func InfluxDBWithTags(r metrics.Registry, d time.Duration, callback Callback, url, database, username, password string, tags map[string]string) {
	u, err := uurl.Parse(url)
	if err != nil {
		log.Printf("unable to parse InfluxDB url %s. err=%v", url, err)
		return
	}

	rep := &reporter{
		reg:      r,
		interval: d,
		callback: callback,
		url:      *u,
		database: database,
		username: username,
		password: password,
		tags:     tags,
	}
	if err := rep.makeClient(); err != nil {
		log.Printf("unable to make InfluxDB client. err=%v", err)
		return
	}

	rep.run()
}

func (r *reporter) makeClient() (err error) {
	r.client, err = client.NewClient(client.Config{
		URL:      r.url,
		Username: r.username,
		Password: r.password,
	})

	return
}

func (r *reporter) run() {
	intervalTicker := time.Tick(r.interval)
	pingTicker := time.Tick(time.Second * 5)

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
