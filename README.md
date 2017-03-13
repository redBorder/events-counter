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
monitor:
  kafka:
    write_topic: "rb_limits"
    read_topics:
      - "rb_counters"
    config:
      bootstrap.servers: "kafka:9092"
      group.id: "monitors"
      topic.auto.offset.reset: "smallest"
  timer_s:
    period: 86400
    offset: 0

counters:
  update_interval_s: 5
  uuid_key: "sensor_uuid"
  kafka:
    write_topic: "rb_limits"
    read_topics:
      - "rb_flow"
    config:
      bootstrap.servers: "kafka:9092"
      group.id: "counters"
      topic.auto.offset.reset: "smallest"

limits:
  uuids:
    - uuid: 2
      type: "bytes"
      limit: 37000000
    - uuid: 4
      type: "messages"
      limit: 1000
    - uuid: 7
      type: "bytes"
      limit: 0
  default:
    type: "messages"
    limit: -1
```
