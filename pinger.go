package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	fastping "github.com/tatsushid/go-fastping"
)

var (
	port = flag.Int("port", 8080, "Port to listen for Prometheus requests")

	avgRtt = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "honestdns_ping_ms",
		Help: "Ping time to HonestDNS in ms",
	})

	pingReceived = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "honestdns_ping_received",
		Help: "Ping responses receiged from HonestDNS",
	})
)

func init() {
	prometheus.MustRegister(pingReceived)
}

func main() {
	flag.Parse()

	var once sync.Once

	http.Handle("/metrics", promhttp.Handler())
	//go func() { log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)) }()

	p := fastping.NewPinger()
	p.AddIP("8.8.8.8")

	var received int
	var duration float64

	p.OnRecv = func(addr *net.IPAddr, rtt time.Duration) {
		received++
		pingReceived.Add(float64(received))
		duration += float64(rtt / time.Millisecond)
	}
	p.OnIdle = func() {
		if received >= 5 {
			once.Do(func() {
				prometheus.MustRegister(avgRtt)
			})
			avgRtt.Set(float64(duration / float64(received)))
			time.Sleep(5 * time.Second)
			received = 0
			duration = 0
		}
	}

	p.RunLoop()
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
