// By Tyler Montgomery, 2015

package main

import (
	//"errors"
	"fmt"
	"github.com/jessevdk/go-flags"
	"os"
	"github.com/op/go-logging"
	"gopkg.in/mcuadros/go-syslog.v2"
	"gopkg.in/redis.v3"
	"encoding/json"
	//"strconv"
	//"strings"
)

const APP_NAME = "gologq"
const APP_VERSION = "0.0.1"

var log = logging.MustGetLogger(APP_NAME)
var format = logging.MustStringFormatter(
	`%{color}%{level:-7s}: %{time} %{shortfile} %{longfunc} %{id:03x}%{color:reset} %{message}`,
)

var opts struct {
	Verbose bool `short:"v" long:"verbose" description:"Enable DEBUG logging"`
	DoVersion bool `short:"V" long:"version" description:"Print version and exit"`

	// Syslog specific options
	ListenAddress string `long:"listen" description:"Syslog receiver host" default:"0.0.0.0"`
	ListenPort int `long:"port" description:"Syslog receiver port" default:"514"`

	// Redis specific options
	RedisHost string `long:"redis_host" description:"Redis host" default:"localhost"`
	RedisPort int `long:"redis_port" description:"Redis port" default:"6379"`
	RedisKey string `long:"redis_key" description:"Redis list key" default:"gologq"`
	RedisPassword string `long:"redis_password" description:"Redis password"`
	RedisDB int64 `long:"redis_db" description:"Redis DB index"`
}

func main() {

	// Parse arguments
	_, err := flags.Parse(&opts)
	if err != nil {
		typ := err.(*flags.Error).Type
		if typ == flags.ErrHelp {
			os.Exit(0)
		} else {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	// Configure logger
	log_backend := logging.NewLogBackend(os.Stderr, "", 0)
	backend_formatter := logging.NewBackendFormatter(log_backend, format)
	logging.SetBackend(backend_formatter)

	if opts.DoVersion == true {
		fmt.Printf("%s %s at your service.\n", APP_NAME, APP_VERSION)
		os.Exit(10)
	}

	if opts.Verbose == true {
		logging.SetLevel(logging.DEBUG, "")
	} else {
		logging.SetLevel(logging.INFO, "")
	}

	hostname, _ := os.Hostname()
	log.Infof("Starting %s version: %s on host %s", APP_NAME, APP_VERSION, hostname)

	// Let's get moving.
	log.Debugf("Commandline options: %+v", opts)
	redis_client := setupRedisClient()
	startServer(redis_client)

	log.Info("Server finished. Exiting.")
}

func setupRedisClient() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", opts.RedisHost, opts.RedisPort),
		Password: opts.RedisPassword,
		DB:       opts.RedisDB,
	})

	pong, err := client.Ping().Result()
	if err != nil {
		log.Fatalf("Unable to contact redis server: %s", err)
	} else {
		log.Debugf("Ping response from redis server: %s", pong)
	}

	return client
}

func handleGoLogs(channel syslog.LogPartsChannel, redis_client *redis.Client) {
	log.Debug("Entered receiver")
	for logParts := range channel {
		json_data, err := json.Marshal(logParts)
		if err != nil {
			log.Errorf("JSON error:", err)
		}
		log.Debugf("RECV: %s", json_data)

		// Push to redis list
		err = redis_client.LPush(opts.RedisKey, fmt.Sprintf("%s", json_data)).Err()
		if err != nil {
			log.Errorf("Redis error: %s", err)
		}
	}
}

func startServer(redis_client *redis.Client) {
	log.Debug("Entered startServer")

	// Start up the syslog server
	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)

	server := syslog.NewServer()
	server.SetFormat(syslog.Automatic)
	server.SetHandler(handler)
	err := server.ListenTCP(fmt.Sprintf("%s:%d", opts.ListenAddress, opts.ListenPort))

	// Were we able to start the server?
	if err != nil {
		log.Fatalf("Can't listen to TCP socket. Failing. %+v", err)
	}

	server.Boot()


	// The main worker here.
	go handleGoLogs(channel, redis_client)

	log.Info("Listening for connections")
	server.Wait()
}

