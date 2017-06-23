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

import rdkafka "github.com/confluentinc/confluent-kafka-go/kafka"

// BootstrapRdKafkaConsumer creates a Kafka consumer.
func BootstrapRdKafkaConsumer(
	attributes map[string]string, topicAttributes map[string]string,
) (*rdkafka.Consumer, error) {

	ta := rdkafka.ConfigMap{
		"auto.offset.reset": "smallest",
	}
	for key, value := range topicAttributes {
		ta.SetKey(key, value)
	}

	a := &rdkafka.ConfigMap{
		"go.events.channel.enable":        true,
		"go.application.rebalance.enable": true,
		"default.topic.config":            ta,
	}

	for key, value := range attributes {
		a.SetKey(key, value)
	}

	consumer, err := rdkafka.NewConsumer(a)
	if err != nil {
		return nil, err
	}

	return consumer, nil
}

// BootstrapRdKafkaProducer creates a Kafka producer.
func BootstrapRdKafkaProducer(attributes map[string]string) (*rdkafka.Producer, error) {

	a := &rdkafka.ConfigMap{}
	for key, value := range attributes {
		a.SetKey(key, value)
	}

	producer, err := rdkafka.NewProducer(a)
	if err != nil {
		return nil, err
	}

	return producer, nil
}
