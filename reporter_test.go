package metflux_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/rcrowley/go-metrics"
	"github.com/sadlil/metflux"
	"github.com/stretchr/testify/assert"
)

func TestReporter(t *testing.T) {
	reg := metrics.NewRegistry()
	config := &metflux.Config{
		InfluxDB: &metflux.InfluxDB{
			URL:      "",
			Database: "test",
			Username: "admin",
			Password: "",
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
			URL:      "",
			Database: "test",
			Username: "admin",
			Password: "",
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
