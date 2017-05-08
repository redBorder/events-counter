[![Build Status](https://travis-ci.org/redBorder/events-counter.svg?branch=master)](https://travis-ci.org/redBorder/events-counter)
[![Coverage Status](https://coveralls.io/repos/github/redBorder/events-counter/badge.svg)](https://coveralls.io/github/redBorder/events-counter)
[![Go Report Card](https://goreportcard.com/badge/github.com/redBorder/events-counter)](https://goreportcard.com/report/github.com/redBorder/events-counter)

# events-counter

* [Overview](#overview)
* [Installing](#installing)
* [Usage](#usage)
* [Configuration](#configuration)

## Overview

`events-counter` is a simple utility that can be used for accounting messages
in a Kafka topic. It supports accounting by number of bytes (regardless the
numer of messages).

Messages are expected to be on JSON format when counting messages.

`events-counter` has two independent modules:

- **UUIDCounter**
  - Reads messages from a set of input topics.
  - Count the number of bytes of these messages.
  - Sends a message periodically to an output topic with the account of bytes
  on the inputs topic.
- **CountersMonitor**
  - The input topic of the counter monitor is the output topic of the UUIDCounter.
  - Reads messages with the account of bytes. Discards old messages.
  - If the number of total bytes exceeds the limit sends an alert.

`events-counter` ignores Teldat sensors traffic, so these kind of sensors can
send an unlimited amount of bytes.  

## License

`events-counter` limits can't be configured. These limits are loaded from a
LICENSE file. These licenses must be signed to be considered a valid license. Also, a
license has an expiration time and it won't be considered valid after
the expiration time.

The public key used to verify the license should be specified at compile time
and can't be modified once the binary is compiled. To specify the public key,
the application should be compiled like this:

```bash
export PUBLIC_KEY="-----BEGIN PUBLIC KEY-----\nMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCqGKukO1De7zhZj6+H0qtjTkVxwTCpvKe4eCZ0\nFPqri0cb2JZfXJ/DgYSF6vUpwmJG8wVQZKjeGcjDOL5UlsuusFncCzWBQ7RKNUSesmQRMSGkVb1/\n3j+skZ6UtW+5u09lHNsj6tQ51s1SPrCBkedbNf0Tp0GbMJDyR4e9T04ZZwIDAQAB\n-----END PUBLIC KEY-----"
make
```

_**Note** that every line break should be **replaced** by "\n"_

## Installing

To install this application ensure you have the
[GOPATH](https://golang.org/doc/code.html#GOPATH) environment variable set and
**[glide](https://glide.sh/)** installed.

```bash
curl https://glide.sh/get | sh
```

And then:

1. Clone this repo and cd to the project

    ```bash
    git clone https://github.com/redBorder/events-counter && cd events-counter
    ```
2. Install dependencies and compile

    ```bash
    make
    ```
3. Install on desired directory

    ```bash
    prefix=/opt/events-counter/ make install
    ```

## Usage

Usage of `events-counter`:

```
--version
    Show version info
--config string
    Config file
--debug
    Print debug info
```

## Configuration

```yaml
counters:
  batch_timeout_s: 5                     # Max time to wait before send a count message
  batch_max_messages: 1000               # Max number of messages to hold before send a count message
  uuid_key: "sensor_uuid"                # JSON key for the UUID
  kafka:                                 # Kafka configuration
    write_topic: "rb_monitor"            # Topic to send the count
    read_topics:                         # Topics to read messages for accounting
      - "rb_flow"
    attributes:                          # Custom internal rdkafka attributes
      bootstrap.servers: "kafka:9092"
      group.id: "counters"

monitor:
  timer:
    period: 86400                        # Width in seconds of the interval between counters reset (86400 -> 24h)
    offset: 0                            # Offset in seconds to change the start of the interval (0 -> 00:00h)
  kafka:                                 # Kafka configuration
    write_topic: "rb_limits"             # Topic to send the alerts
    read_topics:                         # Topics to read messages with accounting info
      - "rb_monitor"
    attributes:                          # Custom internal rdkafka attributes
      bootstrap.servers: "kafka:9092"
      enable.auto.commit: "false"        # IMPORTANT: Should be set to false
      group.id: "monitor"

licenses_directory: /etc/licenses       # Path to the directory where licenses are stored
organization_mode: false                # Ignore organization_uuid. Count every message as belonging to org "*"
```
