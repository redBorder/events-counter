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
	"encoding/json"
	"errors"

	rdkafka "github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/redBorder/events-counter/counter"
	"github.com/redBorder/events-counter/producer"
	"github.com/redBorder/rbforwarder"
	"github.com/redBorder/rbforwarder/components/batch"
)

// UUIDCountersPipeline starts the pipeline for accounting messages
func UUIDCountersPipeline(config *AppConfig) {
	log := log.WithField("prefix", "counter")

	///////////////////////
	// Counters Pipeline //
	///////////////////////

	p, err := BootstrapRdKafkaProducer(config.Counters.Kafka.Attributes)
	if err != nil {
		log.Fatal("Error creating counters producer: " + err.Error())
	}
	factory := producer.NewRdKafkaFactory(p)

	var components []interface{}
	pipeline := rbforwarder.NewRBForwarder(
		rbforwarder.Config{
			Retries:   0,
			Backoff:   0,
			QueueSize: 10,
		})

	components = append(components, &batch.Batcher{
		Config: batch.Config{
			Workers:       1,
			TimeoutMillis: config.Counters.BatchTimeoutSeconds,
			Limit:         uint64(config.Counters.BatchMaxMessages),
		}})
	components = append(components, &counter.Counter{
		Config: counter.Config{
			Workers: 1,
		}})
	components = append(components, &producer.KafkaProducer{
		Config: producer.Config{
			Factory:    factory,
			Workers:    1,
			Topic:      config.Counters.Kafka.WriteTopic,
			Attributes: config.Counters.Kafka.Attributes,
		}})

	pipeline.PushComponents(components)
	pipeline.Run()

	go func() {
		for report := range pipeline.GetReports() {
			if ok := report.(rbforwarder.Report).Code; ok != 0 {
				log.Errorln("UUID Counters error: " + report.(rbforwarder.Report).Status)
			}
		}
	}()

	/////////////////////////
	// Kafka UUID consumer //
	/////////////////////////

	kafkaConsumer, err := BootstrapRdKafkaConsumer(
		config.Counters.Kafka.Attributes, config.Counters.Kafka.TopicAttributes)
	if err != nil {
		log.Fatalln("Error creating Kafka UUID consumer: " + err.Error())
	}

	kafkaConsumer.SubscribeTopics(config.Counters.Kafka.ReadTopics, nil)

	wg.Add(1)
	go func() {
		log.Infof("Started Kafka UUID consumer: (Topics: %v)",
			config.Counters.Kafka.ReadTopics)

	receiving:
		for {
			select {
			case <-terminate:
				log.Debugln("Terminating Kafka UUID consumer...")
				break receiving

			case e := <-kafkaConsumer.Events():
				switch event := e.(type) {
				case rdkafka.AssignedPartitions:
					kafkaConsumer.Assign(event.Partitions)
					log.Debugln(event.String())

				case rdkafka.RevokedPartitions:
					kafkaConsumer.Unassign()
					log.Debugln(event.String())

				case rdkafka.Error:
					log.Errorln(event.String())

				case *rdkafka.Message:
					isTeldat, org, err := checkMessage(config, event.Value)
					if err != nil {
						log.Warn(err)
						continue
					}

					if isTeldat {
						continue
					}

					pipeline.Produce(event.Value, map[string]interface{}{
						// NOTE batch_group and uuid should be the same to ensure that
						// messages from the same organization are grouped together when
						// they are counted.
						"uuid":        org,
						"batch_group": org,
					}, nil)

				default:
					log.Debugln(e.String())
				}
			}
		}

		kafkaConsumer.Close()
		log.Infoln("Kafka UUID consumer finished")
		wg.Done()
	}()
}

func checkMessage(config *AppConfig, message []byte) (
	isTeldat bool, org string, err error) {
	parsed := make(map[string]interface{})
	err = json.Unmarshal(message, &parsed)
	if err != nil {
		return
	}

	if _, ok := parsed["product_name"]; ok {
		isTeldat = true
	}

	if !config.OrganizationMode {
		org = "*"
		return
	}

	if orgIf, ok := parsed["organization_uuid"]; ok {
		org, ok = orgIf.(string)
		if !ok {
			err = errors.New("Field 'organization_uuid' is not string")
			return
		}
	}

	return
}
