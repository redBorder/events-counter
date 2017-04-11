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
	"time"

	rdkafka "github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/redBorder/events-counter/monitor"
	"github.com/redBorder/events-counter/producer"
	"github.com/redBorder/rbforwarder"
)

var reset = make(chan struct{})

// CountersMonitor starts the pipeline for the monitoring process.
func CountersMonitor(config *AppConfig) {
	log := log.WithField("prefix", "monitor")

	wg.Add(1)
	go func() {
		for {
			now := time.Now()
			intervalEnd := monitor.IntervalEndsAt(
				config.Monitor.Timer.Period,
				config.Monitor.Timer.Offset,
				now)
			remaining := intervalEnd.Sub(now)

			limitBytes, err := LoadLicenses(config)
			if err != nil {
				log.Fatalln("Error loading licenses: " + err.Error())
			}

			pipeline := BootstrapMonitorPipeline(config, limitBytes)
			StartConsumingMonitor(pipeline, config)

			log.
				WithField("Time", intervalEnd.String()).
				Infof("Next reset set")

			<-time.After(remaining)

			pipeline.Produce(nil, map[string]interface{}{
				"reset_notification": true,
			}, nil)

			reset <- struct{}{}
		}
	}()
}

// BootstrapMonitorPipeline bootstrap a RBForwarder pipeline
func BootstrapMonitorPipeline(config *AppConfig, limitBytes int64) *rbforwarder.RBForwarder {
	// TODO This only works for one generic UUID
	limits := map[string]uint64{
		"*": uint64(limitBytes),
	}

	p, err := BootstrapRdKafkaProducer(config.Monitor.Kafka.Attributes)
	if err != nil {
		log.Fatal("Error creating monitor producer: " + err.Error())
	}
	mf := producer.NewRdKafkaFactory(p)

	var components []interface{}
	pipeline := rbforwarder.NewRBForwarder(
		rbforwarder.Config{
			Retries:   0,
			Backoff:   0,
			QueueSize: 100,
		})

	components = append(components, &monitor.CountersMonitor{
		Config: monitor.Config{
			Workers: 1,
			Limits:  limits,
			Period:  config.Monitor.Timer.Period,
			Offset:  config.Monitor.Timer.Offset,
			Log:     log,
		}})
	components = append(components, &producer.KafkaProducer{
		Config: producer.Config{
			Factory:    mf,
			Workers:    1,
			Topic:      config.Monitor.Kafka.WriteTopic,
			Attributes: config.Monitor.Kafka.Attributes,
		}})

	pipeline.PushComponents(components)

	return pipeline
}

// StartConsumingMonitor starts receiving kafka messages and sends them to them
// pipeline
func StartConsumingMonitor(pipeline *rbforwarder.RBForwarder, config *AppConfig) {
	log := log.WithField("prefix", "monitor")

	pipeline.Run()

	countersConsumer, err := BootstrapRdKafkaConsumer(
		config.Monitor.Kafka.Attributes, config.Monitor.Kafka.TopicAttributes)
	if err != nil {
		log.Fatalln("Error creating Kafka counters consumer: " + err.Error())
	}

	countersConsumer.SubscribeTopics(config.Monitor.Kafka.ReadTopics, nil)

	go func() {
		for report := range pipeline.GetReports() {
			if ok := report.(rbforwarder.Report).Code; ok != 0 {
				log.Errorln("Monitor error: " + report.(rbforwarder.Report).Status)
			}
		}
	}()

	go func() {
		log.
			WithField("Topics", config.Monitor.Kafka.ReadTopics).
			Infof("Started Kafka Counters consumer")

	receiving:
		for {
			select {
			case <-terminate:
				countersConsumer.Close()
				// Wait for the countersConsumer to be closed or the
				// app could end without finished the closing action.
				wg.Done()
				log.Infoln("Kafka counters consumer finished")
				break receiving

			case <-reset:
				countersConsumer.Close()
				break receiving

			case e := <-countersConsumer.Events():
				switch event := e.(type) {
				case rdkafka.AssignedPartitions:
					countersConsumer.Assign(event.Partitions)
					log.Debugln(event.String())

				case rdkafka.RevokedPartitions:
					countersConsumer.Unassign()
					log.Debugln(event.String())

				case rdkafka.Error:
					log.Errorln(event.String())

				case *rdkafka.Message:
					pipeline.Produce(event.Value, map[string]interface{}{}, nil)

				default:
					log.Debugln(e.String())
				}
			}
		}
	}()
}
