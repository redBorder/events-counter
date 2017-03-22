// Simple utility for counting messages on a Kafka topic.
//
// Copyright (C) 2017 ENEO Tecnologia SL
// Author: Diego Fernández Barrera <bigomby@gmail.com>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"flag"
	"io/ioutil"
	"os"
	"os/signal"
	"sync"

	"github.com/Sirupsen/logrus"
	rdkafka "github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/redBorder/events-counter/counter"
	"github.com/redBorder/events-counter/producer"
	"github.com/redBorder/rbforwarder"
	"github.com/redBorder/rbforwarder/components/batch"
)

var (
	version    string
	configFile string

	wg        = new(sync.WaitGroup)
	terminate = make(chan struct{})
)

func init() {
	versionFlag := flag.Bool("version", false, "Show version info")
	debugFlag := flag.Bool("debug", false, "Show debug info")
	configFlag := flag.String("config", "", "Application configuration file")
	flag.Parse()

	if *versionFlag {
		PrintVersion()
		os.Exit(0)
	}

	if len(*configFlag) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	if *debugFlag {
		logrus.SetLevel(logrus.DebugLevel)
	}

	configFile = *configFlag
}

func main() {
	///////////////////////
	// App Configuration //
	///////////////////////

	rawConfig, err := ioutil.ReadFile(configFile)
	if err != nil {
		logrus.Fatalln("Error reading config file: " + err.Error())
	}

	config, err := ParseConfig(rawConfig)
	if err != nil {
		logrus.Fatalln("Error parsing config: " + err.Error())
	}

	//////////////////
	// rbforwarder //
	//////////////////

	var components []interface{}
	f := rbforwarder.NewRBForwarder(
		rbforwarder.Config{
			Retries:   0,
			Backoff:   0,
			QueueSize: 10,
		})

	components = append(components, &batcher.Batcher{
		Config: batcher.Config{
			Workers:       1,
			TimeoutMillis: config.Counters.BatchTimeoutSeconds,
			Limit:         uint64(config.Counters.BatchMaxMessages),
		},
	})
	components = append(components, &counter.Counter{
		Config: counter.Config{
			Workers: 1,
		},
	})

	p, err := BootstrapRdKafkaProducer(config.Counters.Kafka.Attributes)
	if err != nil {
		logrus.Fatal("Error creating producer: " + err.Error())
	}
	factory := producer.NewRdKafkaFactory(p)
	components = append(components, &producer.KafkaProducer{
		Config: producer.Config{
			Factory:    factory,
			Workers:    1,
			Topic:      config.Counters.Kafka.WriteTopic,
			Attributes: config.Counters.Kafka.Attributes,
		},
	})
	f.PushComponents(components)

	f.Run()

	go func() {
		for report := range f.GetReports() {
			if ok := report.(rbforwarder.Report).Code; ok != 0 {
				logrus.Errorln("Error: " + report.(rbforwarder.Report).Status)
			}
		}
	}()

	/////////////////////////////
	// Kafka counters consumer //
	/////////////////////////////

	countersConsumer, err := BootstrapRdKafkaConsumer(
		config.Counters.Kafka.Attributes, config.Counters.Kafka.TopicAttributes)
	if err != nil {
		logrus.Fatalln("Error creating Kafka consumer: " + err.Error())
	}

	countersConsumer.SubscribeTopics(config.Counters.Kafka.ReadTopics, nil)

	wg.Add(1)
	go func() {
		logrus.Infof("Started Kafka counters consumer: [Topics: %v | Attributes: %v]",
			config.Counters.Kafka.ReadTopics,
			config.Counters.Kafka.Attributes)

	receiving:
		for {
			select {
			case <-terminate:
				logrus.Debugln("Terminating Kafka counters consumer...")
				break receiving

			case e := <-countersConsumer.Events():
				switch event := e.(type) {
				case rdkafka.AssignedPartitions:
					countersConsumer.Assign(event.Partitions)
					logrus.Debugln(event.String())

				case rdkafka.RevokedPartitions:
					countersConsumer.Unassign()
					logrus.Debugln(event.String())

				case rdkafka.Error:
					logrus.Errorln(event.String())

				case *rdkafka.Message:
					// TODO Discard old messages

					f.Produce(event.Value, map[string]interface{}{
						"uuid":        "*",
						"batch_group": "counter",
					}, nil)

				default:
					logrus.Debugln(e.String())
				}
			}
		}

		countersConsumer.Close()
		logrus.Infoln("Kafka counters consumer finished")
		wg.Done()
	}()

	///////////////////
	// Handle SIGINT //
	///////////////////

	sigint := make(chan os.Signal)
	signal.Notify(sigint, os.Interrupt)

	go func() {
		<-sigint
		close(terminate)
	}()

	/////////////
	// The End //
	/////////////

	wg.Wait()
	logrus.Infoln("Bye bye...")
}
