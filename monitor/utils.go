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
	"time"
)

// bootstrapDB creates a new map with the same keys that a given base map, but
// all the values are initialized to zero.
func bootstrapDB(base map[string]uint64) map[string]uint64 {
	db := make(map[string]uint64)
	for k := range base {
		db[k] = 0
	}

	return db
}

// createLimitReachedMessage builds a JSON message alerting that a limit has been
// reached.
func createLimitReachedMessage(uuid string, bytes, limit uint64, timestamp int64) []byte {
	var data []byte

	alert := &Alert{
		Monitor:      "alert",
		Timestamp:    timestamp,
		Type:         "limit_reached",
		UUID:         uuid,
		CurrentBytes: bytes,
		Limit:        limit,
	}

	data, _ = json.Marshal(alert)
	return data
}

// createUknownUUIDMessage builds a JSON message alerting that an UUID does
// not exists on the internal database.
func createUknownUUIDMessage(uuid string) []byte {
	var data []byte

	alert := &Alert{
		Monitor:   "alert",
		Timestamp: time.Now().Unix(),
		Type:      "unknown_uuid",
		UUID:      uuid,
	}

	data, _ = json.Marshal(alert)
	return data
}

// createUknownUUIDMessage builds a JSON message alerting that an UUID does
// not exists on the internal database.
func createResetNotificationMessage() []byte {
	var data []byte

	alert := &Alert{
		Monitor:   "alert",
		Timestamp: time.Now().Unix(),
		Type:      "counters_reset",
	}

	data, _ = json.Marshal(alert)
	return data
}

// checkTimestamp is used to discard old timestamps (messages from previous
// period).
//
// Given a timestamp, will check if the timestamp belongs to an interval of
// "period" seconds width. Also an offset can be specified. It will return true
// if the timestamp belongs to the timestamp, and false in other case.
//
// For example:
//  - "period" is 86400 seconds (24h)
//  - "offset" is 3600 (1h)
//  - "now" is the current timestamp
//  - "timestamp" is the timestamp for 6:00 today
//
//  As the "period" is set to 24h, the function will check if "timestamp"
//  belongs to the current day. The current day starts at 01:00 because "offset"
//  is 1h.
//
func belongsToInterval(timestamp, period, offset, now int64) bool {
	var intervalStart int64
	periodStart := int64(now - now%period)

	if now > periodStart+offset {
		intervalStart = periodStart + offset
	} else {
		intervalStart = periodStart - period + offset
	}

	if timestamp < intervalStart {
		return false
	}

	return true
}

// IntervalEndsAt returns the time when the interval finish
func IntervalEndsAt(period, offset int64, now time.Time) time.Time {
	var intervalEnd int64
	periodStart := int64(now.Unix() - now.Unix()%period)

	if now.Unix() >= periodStart+offset {
		intervalEnd = periodStart + offset + period
	} else {
		intervalEnd = periodStart + offset
	}

	return time.Unix(intervalEnd, 0)
}

// ParseCount gets a json as a map and returns a struct with the values
func ParseCount(data []byte) *Count {
	var ok bool

	count := &Count{}
	msg := make(map[string]interface{})
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil
	}

	monitor, ok := msg["monitor"]
	if !ok {
		return nil
	}

	if count.Monitor, ok = monitor.(string); !ok ||
		count.Monitor != "organization_received_bytes" {
		return nil
	}

	if uuid, ok := msg["organization_uuid"]; ok {
		if count.UUID, ok = uuid.(string); !ok {
			return nil
		}
	}
	if value, ok := msg["value"]; ok {
		floatValue, ok := value.(float64)
		if !ok {
			return nil
		}

		count.Value = uint64(floatValue)
	}
	if timestamp, ok := msg["timestamp"]; ok {
		floatTimestamp, ok := timestamp.(float64)
		if !ok {
			return nil
		}

		count.Timestamp = int64(floatTimestamp)
	}

	return count
}
