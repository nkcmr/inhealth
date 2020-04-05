package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"code.nkcmr.net/sigcancel"
	"github.com/digineo/go-ping"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/cli/v2"
)

type metrics struct {
	mrtt         *prometheus.HistogramVec
	msent, mrecv *prometheus.CounterVec
}

func monitorHost(ctx context.Context, pinger *ping.Pinger, met metrics, host string) {
	log.Printf("monitorHost (host = %s)", host)
	log.Printf("debug: resolving ip address")
	dest, err := net.ResolveIPAddr("ip4", host)
	if err != nil {
		log.Printf("ERROR: failed to resolve host: %s", err)
		return
	}
	log.Printf("pinging %s", dest)
	tck := time.NewTicker(time.Second)
	doPing := func() {
		met.msent.WithLabelValues(host).Inc()
		rtt, err := pinger.Ping(dest, time.Second)
		if err != nil {
			log.Printf("ping error: %s", err)
			return
		}
		met.mrtt.WithLabelValues(host).Observe(rtt.Seconds())
		met.mrecv.WithLabelValues(host).Inc()
	}
	defer tck.Stop()
	for {
		select {
		case <-tck.C:
			go doPing()
		case <-ctx.Done():
			return
		}
	}
}

func run(c *cli.Context) error {
	ctx, cancel := context.WithCancel(context.Background())
	go sigcancel.CancelOnSignal(cancel)
	met := metrics{
		mrtt: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "ping_rtt",
			Help:    "histogram of the rtts of ",
			Buckets: []float64{0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		}, []string{"host"}),
		msent: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "ping_n_sent",
			Help: "counter of sent icmp packets",
		}, []string{"host"}),
		mrecv: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "ping_n_recv",
			Help: "counter of received reply icmp packets",
		}, []string{"host"}),
	}
	prometheus.MustRegister(met.mrtt, met.msent, met.mrecv)
	go func() {
		http.ListenAndServe(":9088", promhttp.Handler())
	}()
	pinger, err := ping.New("0.0.0.0", "")
	if err != nil {
		return errors.Wrap(err, "failed to setup new pinger")
	}
	defer pinger.Close()
	go monitorHost(ctx, pinger, met, "1.0.0.1")
	go monitorHost(ctx, pinger, met, "8.8.8.8")
	go monitorHost(ctx, pinger, met, "icanhazip.com")
	<-ctx.Done()
	log.Printf("wrapping up...")
	return nil
}

func _main(args []string) error {
	app := cli.NewApp()
	app.Name = "inhealth"
	app.Description = "monitor connectivity health using ICMP pings"
	app.Action = run
	return app.Run(args)
}

func main() {
	if err := _main(os.Args); err != nil {
		log.Printf("ERROR: %s", err)
	}
}
