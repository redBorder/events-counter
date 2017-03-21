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
in a Kafka topic. It supports accounting by number of messages or by total
number of bytes (regardless the numer of messages).

Messages are expected to be on JSON format when counting messages.

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
  batch_timeout_s: 5                       # Max time to wait before send a batch
  batch_max_messages: 1000                 # Max number of messages per batch
  uuid_key: "sensor_uuid"                  # JSON key for the UUID
  kafka:                                   # Kafka configuration
    write_topic: "rb_counters"             # Topic to send the count
    read_topics:                           # Topics to read messages for accounting
      - "rb_flow"
    attributes:                            # Custom internal rdkafka attributes
      bootstrap.servers: "kafka:9092"
      group.id: "counters"
```
