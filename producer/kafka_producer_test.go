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
	"testing"

	rdkafka "github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/redBorder/rbforwarder/utils"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

////////////////////
// RdProducerMock //
////////////////////

type RdProducerMock struct {
	mock.Mock
}

func (rdkafkaMock *RdProducerMock) ProduceChannel() chan *rdkafka.Message {
	args := rdkafkaMock.Called()
	return args.Get(0).(chan *rdkafka.Message)
}

func (rdkafkaMock *RdProducerMock) Close() {
	rdkafkaMock.Called()
}

/////////////////
// RdKafkaMock //
/////////////////

type RdKafkaMock struct {
	mock.Mock
}

///////////////
// Done Mock //
///////////////

func (k *RdKafkaMock) NewProducer(config *rdkafka.ConfigMap) (c *RdProducerMock, err error) {
	args := k.Called(config)
	return args.Get(0).(*RdProducerMock), args.Error(1)
}

type Doner struct {
	mock.Mock
	doneCalled chan *utils.Message
}

func (d *Doner) Done(m *utils.Message, code int, status string) {
	d.Called(m, code, status)
	d.doneCalled <- m
}

////////////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////////////

func TestRdKafkaFactory(t *testing.T) {
	Convey("Given a RdKafka Producer", t, func() {
		attributes := &rdkafka.ConfigMap{}
		rdKafka := new(RdKafkaMock)
		mockProducer := new(RdProducerMock)

		rdKafka.On("NewProducer", attributes).Return(mockProducer, nil)
		baseProducer, err := rdKafka.NewProducer(attributes)
		assert.NoError(t, err)

		Convey("When a producer factory is created", func() {
			factory := NewRdKafkaFactory(baseProducer)

			Convey("It should return the base producer", func() {
				producer := factory.CreateProducer()
				So(producer, ShouldEqual, baseProducer)
			})
		})
	})
}

func TestProducer(t *testing.T) {
	Convey("Given working a producer", t, func() {
		kafkaMessages := make(chan *rdkafka.Message, 1)
		rdKafkaProducer := &RdProducerMock{}

		baseProducer := &KafkaProducer{
			Config: Config{
				Workers: 42,
				Factory: NewRdKafkaFactory(rdKafkaProducer),
			},
		}

		producer := baseProducer.Spawn(0)

		Convey("When a message is received by the produced", func() {
			rdKafkaProducer.On("ProduceChannel").Return(kafkaMessages)

			d := new(Doner)
			d.doneCalled = make(chan *utils.Message, 1)
			d.On("Done", mock.AnythingOfType("*utils.Message"), 0, "")

			message := utils.NewMessage()
			message.PushPayload([]byte("Testing"))

			producer.OnMessage(message, d.Done)

			Convey("The message should be sent to Kafka", func() {
				result := <-kafkaMessages

				So(result.Value, ShouldResemble, []byte("Testing"))
				d.AssertExpectations(t)
				rdKafkaProducer.AssertExpectations(t)
			})
		})

		Convey("When a message without payload is received", func() {
			d := new(Doner)
			d.doneCalled = make(chan *utils.Message, 1)
			d.On("Done", mock.AnythingOfType("*utils.Message"), 0, "No payload to produce")

			message := utils.NewMessage()

			Convey("An error getting payload should occurr", func() {
				producer.OnMessage(message, d.Done)
				<-d.doneCalled

				d.AssertExpectations(t)
				rdKafkaProducer.AssertExpectations(t)
			})
		})

		Convey("When a the number of workers is requested", func() {
			workers := producer.Workers()

			Convey("Should be the configured number of worker", func() {
				So(workers, ShouldEqual, 42)
			})
		})
	})
}
