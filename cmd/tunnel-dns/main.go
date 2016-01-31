package main

import (
	"net"
	"os"
	"strconv"

	"github.com/elcuervo/redisurl"
	"github.com/garyburd/redigo/redis"
	"github.com/koding/multiconfig"
	log "gopkg.in/inconshreveable/log15.v2"

	"github.com/pgpst/tunnel/dns"
)

func main() {
	// Load the configuration
	m := multiconfig.NewWithPath(os.Getenv("config"))
	cfg := &dns.Config{}
	m.MustLoad(cfg)

	// Parse the log level
	lvl, err := log.LvlFromString(cfg.LogLevel)
	if err != nil {
		panic(err)
	}

	// Create a new logger
	l := log.New()
	l.SetHandler(
		log.LvlFilterHandler(lvl, log.StdoutHandler),
	)

	// Start the initialization
	il := l.New("module", "init")
	il.Info("Starting up the application")

	// Parse the Redis connection string
	url := redisurl.Parse(cfg.Redis)

	// Create an options list
	opts := []redis.DialOption{}
	if url.Database != 0 {
		opts = append(opts, redis.DialDatabase(url.Database))
	}
	if url.Password != "" {
		opts = append(opts, redis.DialPassword(url.Password))
	}

	// Verify the DNS setup
	results, err := net.LookupNS(cfg.Domain)
	if err != nil {
		il.Warn("Unable to look up the NS of domain", "domain", cfg.Domain, "error", err.Error())
	}
	if len(results) > 0 {
		found := false
		for _, record := range results {
			if record.Host == cfg.Hostname {
				found = true
			}
		}
		if !found {
			existing := []string{}
			for _, record := range results {
				existing = append(existing, record.Host)
			}

			il.Warn("Invalid NS records for the domain", "domain", cfg.Domain, "records", existing)
		}
	} else {
		il.Warn("No NS records found for domain", "domain", cfg.Domain, "error", err.Error())
	}

	// Dial the server
	rc, err := redis.Dial("tcp", url.Host+":"+strconv.Itoa(url.Port), opts...)
	if err != nil {
		il.Crit("Unable to connect to the Redis server", "error", err.Error())
		return
	}

	// Initialize the DNS server
	ds := &dns.DNS{
		Config: cfg,
		Redis:  rc,
	}
	if err := ds.ListenAndServe(); err != nil {
		il.Crit("Error while listening to the specified port", "error", err.Error())
	}
}
