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

package producer

import (
	rdkafka "github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/redBorder/rbforwarder/utils"
)

// RdKafkaProducer is an interface for rdkafka producer. Used for mocking
// purposes.
type RdKafkaProducer interface {
	ProduceChannel() chan *rdkafka.Message
	Close()
}

// RdKafkaFactory is used to create multiple producers with the same
// attributes.
type RdKafkaFactory struct {
	producer RdKafkaProducer
}

// NewRdKafkaFactory returns a new instance of a Producer Factory that always
// returns the same RdKafka Producer.
func NewRdKafkaFactory(producer RdKafkaProducer) *RdKafkaFactory {
	return &RdKafkaFactory{
		producer: producer,
	}
}

// CreateProducer returns the internal RdKafkaProducer.
func (pf *RdKafkaFactory) CreateProducer() RdKafkaProducer {
	return pf.producer
}

// Config contains the configuration for a Counter.
type Config struct {
	Workers    int
	Topic      string
	Attributes map[string]string
	Factory    *RdKafkaFactory
}

// KafkaProducer send the message to a Kafka queue.
type KafkaProducer struct {
	Config

	KafkaProducer RdKafkaProducer
}

// Workers returns the number of workers.
func (kp *KafkaProducer) Workers() int {
	return kp.Config.Workers
}

// Spawn creates a new instance of a Counter worker.
func (kp *KafkaProducer) Spawn(id int) utils.Composer {
	producer := &*kp
	producer.KafkaProducer = kp.Factory.CreateProducer()
	return producer
}

// OnMessage is called when a new message is receive and send the message to
// a Kafka topic.
func (kp *KafkaProducer) OnMessage(m *utils.Message, done utils.Done) {
	payload, err := m.PopPayload()
	if err != nil {
		done(m, 0, "No payload to produce")
		return
	}

	str := string(payload)

	kp.KafkaProducer.ProduceChannel() <- &rdkafka.Message{
		Value: []byte(str),
		TopicPartition: rdkafka.TopicPartition{
			Topic:     &kp.Topic,
			Partition: rdkafka.PartitionAny,
		},
	}

	done(m, 0, "")
}
