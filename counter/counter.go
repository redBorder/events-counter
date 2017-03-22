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
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/redBorder/rbforwarder/utils"
)

// Monitor contains the data for generate a JSON with the count of bytes/messages
type Monitor struct {
	Monitor   string `json:"monitor"`
	Type      string `json:"type"`
	Unit      string `json:"unit"`
	Value     uint64 `json:"value"`
	UUID      string `json:"uuid"`
	Timestamp int64  `json:"timestamp"`
}

// Config contains the configuration for a Counter
type Config struct {
	Workers int
}

// Counter counts the number of messages sents
type Counter struct {
	Config
}

// Workers returns the number of workers
func (c *Counter) Workers() int {
	return c.Config.Workers
}

// Spawn creates a new instance of a Counter worker
func (c *Counter) Spawn(id int) utils.Composer {
	return &*c
}

// OnMessage is called when a new message is receive. Counts the number of
// bytes on the message (or messages if a batch is received) and send a JSON
// formatted message to the next component.
func (c *Counter) OnMessage(m *utils.Message, done utils.Done) {
	payload, err := m.PopPayload()
	if err != nil {
		logrus.Errorln(err)
		done(m, 1, "Can't get payload from message")
		return
	}

	countData := Monitor{
		Monitor:   "data",
		Type:      "counter",
		Unit:      "bytes",
		Value:     uint64(len(payload)),
		Timestamp: time.Now().Unix(),
	}

	if !m.Opts.Has("uuid") {
		done(m, 101, "No uuid found")
		return
	}

	if uuid, ok := m.Opts.Get("uuid"); ok {
		if uuid, ok := uuid.(string); ok {
			countData.UUID = uuid
		}
	}

	countMessage, _ := json.Marshal(countData)

	m.PushPayload(countMessage)
	done(m, 0, "")
}
