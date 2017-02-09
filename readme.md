# gologq
A simple and easy way to send data from rsyslog locally to a redis server.
Pronounced 'go log queue'.

## Why would you want this?
If you're using Logstash to ship events from rsyslog to an Elasticsearch server, consider this scenario:
- You have a large amount of servers, with resource restrictions (not much free RAM)
- You want to have a central Logstash server to handle all parsing of events (single configuration file to manage)
- You don't want to risk downtime of Logstash preventing your log entries from making their way to Elasticsearch

In this case, you'd implement a pipeline of sorts using Redis as a queue for all log events.

Redis' protocol is very easy to speak (versus something like RabbitMQ, or even Kafka), and it's easy to host.
If you're using AWS, you can get a ElastiCache server in a few clicks.

You simply ship log events directly to Redis (in my case, NodeJS Winston Redis output for applications and gologq for syslog), Logstash picks up the events
and parses them, then ships them to Elasticsearch.

With this system, if Logstash is unavailable to parse the events they simply queue up in Redis until Logstash returns.

## Why not use omhiredis from rsyslog?
Good point! Amazon Linux and CentOS don't include a new enough version of rsyslog, and omhiredis wasn't easily available.

The easiest path forward for me was to write a quick service in Golang to push events to Redis so Logstash could pick them up.

## Great, how do I use this?
1. Download a release from this GitHub page
2. Extract the .tar.gz and copy the gologq binary to `/usr/local/bin/gologq`
3. Copy the `docs/30-gologq-output.conf` rsyslog config file to `/etc/rsyslog.d/30-gologq-output.conf`
4. If you're using systemd, you can also copy the included systemd service file to `/etc/systemd/system/gologq.service` and create a `/etc/sysconfig/gologq`
file to store the configuration environment variables (one per line, `KEY=value` format).

Configuration options can be specified as environment variables using the `GOLOGQ_` variant of the CLI option (see help for full list).

## Sample Logstash Config
The `docs/sample-logstash.conf` file contains a sample Logstash configuration that can be used with Gologq.

By default it will split events from HAProxy into their own index.
You'll need to define your own `HAPROXYHTTPBASE` pattern, however.

## Compiling gologq yourself
This project was tested only on Go 1.6.x . If you're using Go 1.5.x you'll need to enable `GO15VENDOREXPERIMENT`.

* If you've got yourself a `GOPATH` set up, simply run `go get github.com/thecubed/gologq`
  and you'll get a binary in your `$GOPATH/bin` folder.
* If you don't have a `GOPATH` set up, it's easy. Just `mkdir ~/go && export GOPATH=$HOME/go && go get github.com/thecubed/gologq`.
  Once that's done, you'll have a `gologq` binary in `~/go/bin/` ready to go!

This project uses govendor to handle vendoring dependencies.
I've modified Jeromer's syslogparser for rfc5424 to support longer `app_name` values from rsyslog.

## Program Help
```
Usage:
  gologq [OPTIONS]

Application Options:
  -v, --verbose         Enable DEBUG logging [$GOLOGQ_VERBOSE]
  -V, --version         Print version and exit
      --listen=         Syslog receiver host (default: 0.0.0.0) [$GOLOGQ_LISTEN_ADDR]
      --port=           Syslog receiver port (default: 514) [$GOLOGQ_LISTEN_PORT]
      --redis_host=     Redis host (default: localhost) [$GOLOGQ_REDIS_HOST]
      --redis_port=     Redis port (default: 6379) [$GOLOGQ_REDIS_PORT]
      --redis_key=      Redis list key (default: gologq) [$GOLOGQ_REDIS_KEY]
      --redis_password= Redis password [$GOLOGQ_REDIS_PASSWORD]
      --redis_db=       Redis DB index [$GOLOGQ_REDIS_DB]
      --workers=        Number of worker threads to spawn (Default: CPUs * 3) [$GOLOGQ_NUM_WORKERS]

Help Options:
  -h, --help            Show this help message
```

## Sample Program Output
```
INFO   : 2017-01-30T21:34:31.309Z main.go:94 main 001 Starting gologq version: 0.0.1 on host mesos-slave-i-1be5fe98
INFO   : 2017-01-30T21:34:31.31Z main.go:165 startServer 002 Spawning 24 worker threads...
INFO   : 2017-01-30T21:34:31.31Z main.go:170 startServer 003 Listening for connections
```

## Redis Output Format
Gologq outputs parsed syslog entries in JSON format to a Redis list. The fields are defined by rfc5424:
```
{
   "app_name":"myapp[13393]:",
   "client":"127.0.0.1:56560",
   "facility":16,
   "hostname":"mesos-master-1",
   "message":"This is a test message from rsyslog!",
   "msg_id":"-",
   "priority":132,
   "proc_id":"13393",
   "severity":4,
   "structured_data":"-",
   "timestamp":"2017-02-09T00:28:09.196916Z",
   "tls_peer":"",
   "version":0
}
```

