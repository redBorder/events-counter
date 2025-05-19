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
	"github.com/benbjohnson/clock"
	"github.com/redBorder/rbforwarder/utils"
	log "github.com/sirupsen/logrus"
)

type logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Infoln(args ...interface{})
}

type nullLogger struct{}

func (n *nullLogger) Debugf(format string, args ...interface{}) {}
func (n *nullLogger) Infof(format string, args ...interface{})  {}
func (n *nullLogger) Infoln(args ...interface{})                {}

// Config contains the configuration for a Monitor.
type Config struct {
	Limits  map[string]uint64
	Period  int64
	Offset  int64
	Workers int
	Log     logger

	clk clock.Clock
}

// CountersMonitor process count messages and check if the maximum of allowed messages
// has been reached.
type CountersMonitor struct {
	db map[string]uint64
	Config
}

// Workers returns the number of workers.
func (mon *CountersMonitor) Workers() int {
	return mon.Config.Workers
}

// Spawn creates a new instance of a Monitor.
func (mon *CountersMonitor) Spawn(id int) utils.Composer {
	monitor := &*mon
	if monitor.clk == nil {
		monitor.clk = clock.New()
	}
	if monitor.Log == nil {
		monitor.Log = new(nullLogger)
	}
	monitor.db = bootstrapDB(mon.Limits)

	return monitor
}

// OnMessage process new messages.
//   - Parses the JSON message and check if the UUID is on the limits database,
//     if not, a message alerting an unknown uuid is sent to kafka.
//   - If the UUID is known (is on the limit map), increment the count of messages
//     on the internal database.
//   - Check if the updated value exceds the allowed number of bytes and if it
//     does, send an alert to Kafka.
func (mon *CountersMonitor) OnMessage(m *utils.Message, done utils.Done) {
	var (
		payload []byte
		err     error
		bytes   uint64
	)

	if showTotal, ok := m.Opts.Get("show_total"); ok {
		if showTotalBool, ok := showTotal.(bool); ok {
			if showTotalBool {
				var total uint64 = 0

				for k, v := range mon.db {
					total += v
					mon.Log.Infof("[%s] Consumed bytes %d", k, v)
				}
				// Imprime el total
				log.Infof("Total consumed bytes: %d", total)
			}
		}

		done(m, 0, "Show total")
		return
	}

	if _, ok := m.Opts.Get("allowed_licenses"); ok {
		if resetCounters, ok := m.Opts.Get("reset_counters"); ok {
			if shouldReset, ok := resetCounters.(bool); ok {
				if shouldReset {
					for organization := range mon.db {
						mon.db[organization] = 0
					}
					mon.Log.Infoln("Counters has been reset")
				}
			}
		}

		licenses, _ := m.Opts.Get("licenses")
		// FIXME check assertion
		m.PushPayload(createLicensesAllowedMessage(licenses.([]string)))
		done(m, 0, "Allowed licenses")
		return
	}

	if payload, err = m.PopPayload(); err != nil {
		done(m, 0, "No payload to produce")
		return
	}

	count := ParseCount(payload)
	if count == nil {
		done(m, 0, "Not counter message")
		return
	}

	if ok := belongsToInterval(count.Timestamp, mon.Period, mon.Offset, mon.clk.Now().Unix()); !ok {
		done(m, 0, "Message too old")
		return
	}

	var ok bool
	if bytes, ok = mon.db[count.UUID]; !ok {
		m.PushPayload(createUknownUUIDMessage(count.UUID))
		done(m, 0, "Unknown UUID: \""+count.UUID+"\"")
		return
	}
	bytes += count.Value
	mon.db[count.UUID] = bytes

	if bytes < mon.Limits[count.UUID] {
		done(m, 0, "Limit not reached")
		return
	}

	mon.Log.Debugf("Sensor %s has reached the limit (%d)", count.UUID, bytes)
	m.PushPayload(
		createLimitReachedMessage(count.UUID, bytes, mon.Limits[count.UUID], mon.clk.Now().Unix()),
	)
	done(m, 0, "Limit reached")
}
