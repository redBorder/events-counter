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

package monitor

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/redBorder/rbforwarder/utils"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/mock"
)

type Doner struct {
	mock.Mock
	doneCalled chan *utils.Message
}

func (d *Doner) Done(m *utils.Message, code int, status string) {
	d.Called(m, code, status)
	d.doneCalled <- m
}

///////////
// TESTS //
///////////

func TestMonitor(t *testing.T) {
	Convey("Given a monitor component", t, func() {
		base := &CountersMonitor{
			Config: Config{
				Workers: 42,
				Period:  60,
				Offset:  0,
				Limits: map[string]uint64{
					"my_uuid":      100,
					"another_uuid": 20,
				},
				clk: clock.NewMock(),
			},
		}
		base.clk.(*clock.Mock).Set(time.Unix(643975200, 0))
		monitor := base.Spawn(1)

		Convey("When a monitor is spawned", func() {
			Convey("Should not be the same object as his base", func() {
				So(&base, ShouldNotEqual, &monitor)
			})
		})

		Convey("When a the number of workers is requested", func() {
			workers := monitor.Workers()

			Convey("Should be the configured number of worker", func() {
				So(workers, ShouldEqual, 42)
			})
		})

		Convey("When a message without payload is received", func() {
			d := new(Doner)
			d.doneCalled = make(chan *utils.Message, 1)
			d.On("Done", mock.AnythingOfType("*utils.Message"), 0, "No payload to produce")

			message := utils.NewMessage()

			Convey("Message should be ignored", func() {
				monitor.OnMessage(message, d.Done)
				result := <-d.doneCalled
				data, err := result.PopPayload()
				So(err, ShouldNotBeNil)
				So(data, ShouldBeNil)

				d.AssertExpectations(t)
			})
		})

		Convey("When a message is received from a unknow UUID", func() {
			message := utils.NewMessage()
			message.PushPayload([]byte(`
				{
					"monitor":"organization_received_bytes",
					"unit":"bytes",
					"value":10,
					"uuid":"unknown",
					"timestamp":643975200
				}
			`))

			Convey("Should alert the unknown UUID", func() {
				d := new(Doner)
				d.doneCalled = make(chan *utils.Message, 1)
				d.On("Done", mock.AnythingOfType("*utils.Message"), 0, "Unknown UUID: \"unknown\"")

				monitor.OnMessage(message, d.Done)
				result := <-d.doneCalled
				data, err := result.PopPayload()
				So(err, ShouldBeNil)

				response := make(map[string]interface{})
				err = json.Unmarshal(data, &response)
				So(err, ShouldBeNil)

				So(response["monitor"], ShouldEqual, "alert")
				So(response["type"], ShouldEqual, "unknown_uuid")
				So(response["uuid"], ShouldEqual, "unknown")
				d.AssertExpectations(t)
			})
		})

		Convey("When a message is received from a Teldat sensor", func() {
			message := utils.NewMessage()
			message.PushPayload([]byte(`
				{
					"monitor":"organization_received_bytes",
					"unit":"bytes",
					"value":10,
					"uuid":"*",
					"timestamp":643975200,
					"is_teldat":true
				}
			`))

			Convey("Should ignore the message", func() {
				d := new(Doner)
				d.doneCalled = make(chan *utils.Message, 1)
				d.On("Done", mock.AnythingOfType("*utils.Message"), 0, mock.AnythingOfType("string"))

				monitor.OnMessage(message, d.Done)
				result := <-d.doneCalled
				_, err := result.PopPayload()
				So(err, ShouldNotBeNil)

				d.AssertExpectations(t)
			})
		})

		Convey("When a message with an unknown monitor is received", func() {
			message := utils.NewMessage()
			message.PushPayload([]byte(`
				{
					"monitor":"cpu",
					"unit":"%",
					"value":50,
					"timestamp":643975200
				}
			`))

			Convey("Should discard the message", func() {
				d := new(Doner)
				d.doneCalled = make(chan *utils.Message, 1)
				d.On("Done",
					mock.AnythingOfType("*utils.Message"), 0, mock.AnythingOfType("string"),
				)

				monitor.OnMessage(message, d.Done)
				result := <-d.doneCalled
				data, err := result.PopPayload()
				So(err, ShouldNotBeNil)
				So(data, ShouldBeNil)

				d.AssertExpectations(t)
			})
		})

		Convey("When a invalid (no json) message is received", func() {
			message := utils.NewMessage()
			message.PushPayload([]byte("not a json message"))
			message.Opts.Set("uuid", "test_uuid")

			Convey("Should error", func() {
				d := new(Doner)
				d.doneCalled = make(chan *utils.Message, 1)
				d.On("Done", mock.AnythingOfType("*utils.Message"), 0, mock.AnythingOfType("string"))

				monitor.OnMessage(message, d.Done)
				result := <-d.doneCalled
				data, err := result.PopPayload()
				So(err, ShouldNotBeNil)
				So(data, ShouldBeNil)

				d.AssertExpectations(t)
			})
		})

		Convey("When a message is received with an old timestamp", func() {
			message := utils.NewMessage()
			message.PushPayload([]byte(`
				{
					"monitor":"organization_received_bytes",
					"unit":"bytes",
					"value":10,
					"uuid":"unknown",
					"timestamp":643975200
				}`))

			base.clk.(*clock.Mock).Add(61 * time.Second)

			Convey("Should be discarded", func() {
				d := new(Doner)
				d.doneCalled = make(chan *utils.Message, 1)
				d.On("Done", mock.AnythingOfType("*utils.Message"), 0, mock.AnythingOfType("string"))

				monitor.OnMessage(message, d.Done)
				result := <-d.doneCalled
				data, err := result.PopPayload()
				So(err, ShouldNotBeNil)
				So(data, ShouldBeNil)

				d.AssertExpectations(t)
			})
		})
	})
}

func TestMonitorReset(t *testing.T) {
	base := &CountersMonitor{
		Config: Config{
			Workers: 42,
			Period:  60,
			Offset:  0,
			Limits: map[string]uint64{
				"my_uuid":      100,
				"another_uuid": 20,
			},
			clk: clock.NewMock(),
		},
	}
	base.clk.(*clock.Mock).Set(time.Unix(643975200, 0))
	monitor := base.Spawn(1)

	Convey("Given two messages", t, func() {
		Convey("When messages are received", func() {
			Convey("The first one should not trigger the alert", func() {
				d := new(Doner)
				d.doneCalled = make(chan *utils.Message, 1)
				d.On("Done", mock.AnythingOfType("*utils.Message"), 0, mock.AnythingOfType("string"))

				message := utils.NewMessage()
				message.PushPayload([]byte(`
					{
					"monitor":"organization_received_bytes",
					"unit":"bytes",
					"value":51,
					"uuid":"my_uuid",
					"timestamp":643975200
					}`))

				monitor.OnMessage(message, d.Done)
				result := <-d.doneCalled
				data, err := result.PopPayload()
				So(err, ShouldNotBeNil)
				So(data, ShouldBeNil)

				bytes := monitor.(*CountersMonitor).db["my_uuid"]
				So(bytes, ShouldEqual, 51)

				d.AssertExpectations(t)
			})

			Convey("The second one should trigger the alert", func() {
				message := utils.NewMessage()
				message.PushPayload([]byte(`
					{
					"monitor":"organization_received_bytes",
					"unit":"bytes",
					"value":51,
					"uuid":"my_uuid",
					"timestamp":643975200
					}`))

				d := new(Doner)
				d.doneCalled = make(chan *utils.Message, 1)
				d.On("Done", mock.AnythingOfType("*utils.Message"), 0, mock.AnythingOfType("string"))

				monitor.OnMessage(message, d.Done)
				result := <-d.doneCalled
				data, err := result.PopPayload()
				So(data, ShouldNotBeNil)
				So(err, ShouldBeNil)

				response := make(map[string]interface{})
				err = json.Unmarshal(data, &response)
				So(err, ShouldBeNil)
				So(response["monitor"], ShouldEqual, "alert")
				So(response["type"], ShouldEqual, "limit_reached")
				So(response["uuid"], ShouldEqual, "my_uuid")

				bytes := monitor.(*CountersMonitor).db["my_uuid"]
				So(bytes, ShouldEqual, 102)

				d.AssertExpectations(t)
			})
		})

		Convey("When a reset message is received", func() {
			Convey("Should send a reset notification", func() {
				message := utils.NewMessage()
				message.Opts.Set("reset_notification", nil)

				d := new(Doner)
				d.doneCalled = make(chan *utils.Message, 1)
				d.On("Done", mock.AnythingOfType("*utils.Message"), 0, mock.AnythingOfType("string"))

				monitor.OnMessage(message, d.Done)
				result := <-d.doneCalled
				data, err := result.PopPayload()
				So(data, ShouldNotBeNil)
				So(err, ShouldBeNil)

				response := make(map[string]interface{})
				err = json.Unmarshal(data, &response)
				So(err, ShouldBeNil)
				So(response["monitor"], ShouldEqual, "alert")
				So(response["type"], ShouldEqual, "counters_reset")

				bytes := monitor.(*CountersMonitor).db["my_uuid"]
				So(bytes, ShouldEqual, 0)

				d.AssertExpectations(t)
			})

			Convey("Should be able to send more messages after timeout resets", func() {
				message := utils.NewMessage()
				message.PushPayload([]byte(`
						{
						"monitor":"organization_received_bytes",
						"unit":"bytes",
						"value":51,
						"uuid":"my_uuid",
						"timestamp":643975200
						}`))

				d := new(Doner)
				d.doneCalled = make(chan *utils.Message, 1)
				d.On("Done", mock.AnythingOfType("*utils.Message"), 0, mock.AnythingOfType("string"))

				monitor.OnMessage(message, d.Done)
				result := <-d.doneCalled
				data, err := result.PopPayload()
				So(data, ShouldBeNil)
				So(err, ShouldNotBeNil)

				bytes := monitor.(*CountersMonitor).db["my_uuid"]
				So(bytes, ShouldEqual, 51)

				d.AssertExpectations(t)
			})
		})
	})
}

func TestMockClock(t *testing.T) {
	Convey("Given a base for spawining Counters Monitor instances", t, func() {
		base := &CountersMonitor{
			Config: Config{
				Workers: 42,
				Period:  60,
				Offset:  0,
			},
		}
		monitor := base.Spawn(1)

		Convey("When no clock is set on the configuration", func() {
			Convey("The internal clock should be a real clock", func() {
				So(monitor.(*CountersMonitor).clk, ShouldNotBeNil)
			})
		})
	})
}
