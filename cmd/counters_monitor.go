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

// CountersMonitor starts the pipeline for the monitoring process.
func CountersMonitor(config *AppConfig) {
	log := log.WithField("prefix", "monitor")

	///////////////////////
	// Monitors Pipeline //
	///////////////////////

	limits := make(map[string]uint64)
	for _, uuidLimit := range config.Limits {
		limits[uuidLimit.UUID] = uint64(uuidLimit.Limit)
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
	pipeline.Run()

	go func() {
		for report := range pipeline.GetReports() {
			if ok := report.(rbforwarder.Report).Code; ok != 0 {
				log.Errorln("Monitor error: " + report.(rbforwarder.Report).Status)
			}
		}
	}()

	/////////////////////////
	// Reset notifications //
	/////////////////////////

	go func() {
		for {
			now := time.Now()
			intervalEnd := monitor.IntervalEndsAt(
				config.Monitor.Timer.Period,
				config.Monitor.Timer.Offset,
				now)
			remaining := intervalEnd.Sub(now)

			log.Infof("Next reset will occur at: %s", intervalEnd)
			<-time.After(remaining)

			pipeline.Produce(nil, map[string]interface{}{
				"reset_notification": true,
			}, nil)
			log.Infoln("All counters are set to 0")
		}
	}()

	/////////////////////////////
	// Kafka counters consumer //
	/////////////////////////////

	countersConsumer, err := BootstrapRdKafkaConsumer(
		config.Monitor.Kafka.Attributes, config.Monitor.Kafka.TopicAttributes)
	if err != nil {
		log.Fatalln("Error creating Kafka counters consumer: " + err.Error())
	}

	countersConsumer.SubscribeTopics(config.Monitor.Kafka.ReadTopics, nil)

	wg.Add(1)
	go func() {
		log.Infof("Started Kafka Counters consumer: (Topics: %v)",
			config.Monitor.Kafka.ReadTopics)

	receiving:
		for {
			select {
			case <-terminate:
				log.Debugln("Terminating Kafka counters consumer...")
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

		countersConsumer.Close()
		log.Infoln("Kafka counters consumer finished")
		wg.Done()
	}()
}
