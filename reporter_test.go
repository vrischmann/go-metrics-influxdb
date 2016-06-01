package metflux_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/influxdata/influxdb/client"
	"github.com/rcrowley/go-metrics"
	"github.com/sadlil/metflux"
	"github.com/stretchr/testify/assert"
)

func TestReporter(t *testing.T) {
	reg := metrics.NewRegistry()
	config := &metflux.Config{
		InfluxDB: &metflux.InfluxDB{
			URL:      "http://104.154.115.200:8086",
			Database: "test",
			Username: "admin",
			Password: "seeG9QN6U4isAp3y",
		},
		Registry: reg,
		Interval: time.Second * 5,
	}

	reporter, err := metflux.New(config)
	assert.Nil(t, err)

	reporter.Run()

	for i := 1; i <= 20; i++ {
		con1 := reg.GetOrRegister("conn1", metrics.NewCounter()).(metrics.Counter)
		con2 := reg.GetOrRegister("conn2", metrics.NewCounter()).(metrics.Counter)

		con1.Inc(1)
		con2.Inc(5)

		time.Sleep(time.Second * 10)
	}
}

func TestReporterWithCallback(t *testing.T) {
	reg := metrics.NewRegistry()
	config := &metflux.Config{
		InfluxDB: &metflux.InfluxDB{
			URL:      "http://104.154.115.200:8086",
			Database: "test",
			Username: "admin",
			Password: "seeG9QN6U4isAp3y",
		},
		Registry: reg,
		Interval: time.Second * 5,
		Callbacks: []metflux.Callback{
			func(i interface{}) {
				fmt.Println("inside callback")
			},
		},
	}

	reporter, err := metflux.New(config)
	assert.Nil(t, err)

	reporter.Run()

	for i := 1; i <= 20; i++ {
		con1 := reg.GetOrRegister("conn1", metrics.NewCounter()).(metrics.Counter)
		con2 := reg.GetOrRegister("conn2", metrics.NewCounter()).(metrics.Counter)

		con1.Inc(1)
		con2.Inc(5)

		time.Sleep(time.Second * 10)
	}
}

type CustomMetricType struct {
	cn int64
}

type CustomMetricInterface interface {
	Inc()
	Read() int64
}

func (c *CustomMetricType) Inc() {
	c.cn++
}

func (c *CustomMetricType) Read() int64 {
	val := c.cn
	c.cn--
	return val
}

var process = func(i interface{}) client.Point {
	m, _ := i.(CustomMetricInterface)
	fmt.Println("reading custom metric")
	return client.Point{
		Measurement: "custom-metirc",
		Fields: map[string]interface{}{
			"value": m.Read(),
		},
		Time: time.Now(),
	}
}

func TestReporterWithCustomMetric(t *testing.T) {
	reg := metrics.NewRegistry()

	config := &metflux.Config{
		InfluxDB: &metflux.InfluxDB{
			URL:      "http://104.154.115.200:8086",
			Database: "test",
			Username: "admin",
			Password: "seeG9QN6U4isAp3y",
		},
		Registry: reg,
		Interval: time.Second * 5,

		CustomMetrics: []metflux.CustomMetric{
			metflux.NewCustomMetric(func(i interface{}) bool { return true }, process),
		},
	}

	reporter, err := metflux.New(config)
	assert.Nil(t, err)

	reporter.Run()

	for i := 1; i <= 20; i++ {
		fmt.Println(">>> ")
		con1 := reg.GetOrRegister("conn-custom", metrics.NewCounter()).(metrics.Counter)
		con1.Inc(2)
		time.Sleep(time.Second * 10)
	}
}
