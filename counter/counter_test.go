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

package counter

import (
	"encoding/json"
	"testing"

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

func TestCounter(t *testing.T) {
	Convey("Given a counter component", t, func() {
		factory := Counter{
			Config{Workers: 42}}

		counter := factory.Spawn(1)

		Convey("When a valid message is received with UUID", func() {
			message := utils.NewMessage()
			message.PushPayload([]byte(`Lorem ipsum dolor sit amet, consectetur adipisicing
        elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.
        Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi
        ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit
        in voluptate velit esse cillum dolore eu fugiat nulla pariatur.
        Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia
        deserunt mollit anim id est laborum.`))
			message.Opts.Set("uuid", "test_uuid")

			Convey("A JSON with the number of bytes should be generated", func() {
				d := new(Doner)
				d.doneCalled = make(chan *utils.Message, 1)
				d.On("Done", mock.AnythingOfType("*utils.Message"), 0, "")

				counter.OnMessage(message, d.Done)
				result := <-d.doneCalled
				payload, err := result.PopPayload()
				So(err, ShouldBeNil)

				monitor := Monitor{}
				err = json.Unmarshal(payload, &monitor)
				So(err, ShouldBeNil)

				So(monitor.Monitor, ShouldEqual, "data")
				So(monitor.Type, ShouldEqual, "counter")
				So(monitor.Unit, ShouldEqual, "bytes")
				So(monitor.Value, ShouldEqual, 494)

				d.AssertExpectations(t)
			})
		})

		Convey("When a valid message is received without UUID", func() {
			message := utils.NewMessage()
			message.PushPayload([]byte("Testing"))

			Convey("A missing UUID error should occurr", func() {
				d := new(Doner)
				d.doneCalled = make(chan *utils.Message, 1)
				d.On("Done", mock.AnythingOfType("*utils.Message"), 101, "No uuid found")

				counter.OnMessage(message, d.Done)
				<-d.doneCalled

				d.AssertExpectations(t)
			})
		})

		Convey("When a message without payload is received", func() {
			message := utils.NewMessage()

			Convey("An error getting payload should occurr", func() {
				d := new(Doner)
				d.doneCalled = make(chan *utils.Message, 1)
				d.On("Done", mock.AnythingOfType("*utils.Message"), 0, "No payload to produce")

				counter.OnMessage(message, d.Done)
				<-d.doneCalled

				d.AssertExpectations(t)
			})
		})

		Convey("When a the number of workers is requested", func() {
			workers := counter.Workers()

			Convey("Should be the configured number of worker", func() {
				So(workers, ShouldEqual, 42)
			})
		})
	})
}
