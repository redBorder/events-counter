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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBootstrapDB(t *testing.T) {
	var exists bool
	var value uint64

	db := bootstrapDB(
		map[string]uint64{
			"key1": 1234,
			"key2": 4321,
			"key3": 0000,
		})

	value, exists = db["key1"]
	assert.True(t, exists)
	assert.Zero(t, value)

	value, exists = db["key2"]
	assert.True(t, exists)
	assert.Zero(t, value)

	value, exists = db["key3"]
	assert.True(t, exists)
	assert.Zero(t, value)

	_, exists = db["key4"]
	assert.False(t, exists)
}

func TestCreateLimitReachedMessage(t *testing.T) {
	expected := `{` +
		`"monitor":"alert",` +
		`"type":"limit_reached",` +
		`"uuid":"my_uuid",` +
		`"current_bytes":120,` +
		`"limit":200,` +
		`"timestamp":12345678}`

	message := createLimitReachedMessage("my_uuid", 120, 200, 12345678)
	assert.Equal(t, expected, string(message))
}

func TestCheckTimestampBelongsToInterval1(t *testing.T) {
	var (
		timestamp int64 = 643975200 // (GMT): Tue, 29 May 1990 10:00:00 GMT
		now       int64 = 643993200 // (GMT): Tue, 29 May 1990 15:00:00 GMT
		period    int64 = 86400     // 24h
		offset    int64             // 0s
	)

	belongsTo := belongsToInterval(timestamp, period, offset, now)
	assert.True(t, belongsTo)
}

func TestCheckTimestampBelongsToInterval2(t *testing.T) {
	var (
		timestamp int64 = 643975200 // (GMT): Tue, 29 May 1990 10:00:00 GMT
		now       int64 = 644027400 // (GMT): Tue, 29 May 1990 00:30:00 GMT
		period    int64 = 86400     // 24h
		offset    int64 = 3600      // 1h
	)

	belongsTo := belongsToInterval(timestamp, period, offset, now)
	assert.True(t, belongsTo)
}

func TestCheckTimestampNotBelongsToInterval1(t *testing.T) {
	var (
		timestamp int64 = 643975200 // (GMT): Tue, 29 May 1990 10:00:00 GMT
		now       int64 = 644027400 // (GMT): Tue, 30 May 1990 00:30:00 GMT
		period    int64 = 86400     // 24h
		offset    int64             // 0s
	)

	belongsTo := belongsToInterval(timestamp, period, offset, now)
	assert.False(t, belongsTo)
}

func TestCheckTimestampNotBelongsToInterval2(t *testing.T) {
	var (
		timestamp int64 = 643975200 // (GMT): Tue, 29 May 1990 10:00:00 GMT
		now       int64 = 644031000 // (GMT): Tue, 30 May 1990 01:30:00 GMT
		period    int64 = 86400     // 24h
		offset    int64 = 3600      // 1h
	)

	belongsTo := belongsToInterval(timestamp, period, offset, now)
	assert.False(t, belongsTo)
}

func TestIntervalEndsAt1(t *testing.T) {
	var (
		now      int64 = 1490259439 // (GMT): Thu, 23 Mar 2017 08:57:19 GMT
		expected int64 = 1490259600 // (GMT): Thu, 23 Mar 2017 09:00:00 GMT
		period   int64 = 300        // 5m
		offset   int64              // 0s
	)

	intervalEnd := IntervalEndsAt(period, offset, time.Unix(now, 0))
	assert.Equal(t, expected, intervalEnd.Unix())
}

func TestIntervalEndsAt2(t *testing.T) {
	var (
		now      int64 = 1490229000 // (GMT): Thu, 23 Mar 2017 00:30:00 GMT
		expected int64 = 1490230800 // (GMT): Thu, 23 Mar 2017 01:00:00 GMT
		period   int64 = 86400      // 24h
		offset   int64 = 3600       // 1h
	)

	intervalEnd := IntervalEndsAt(period, offset, time.Unix(now, 0))
	assert.Equal(t, expected, intervalEnd.Unix())
}

func TestIntervalEndsAt3(t *testing.T) {
	var (
		now      int64 = 1490259600 // (GMT): Thu, 23 Mar 2017 09:00:00 GMT
		expected int64 = 1490259900 // (GMT): Thu, 23 Mar 2017 09:05:00 GMT
		period   int64 = 300        // 5m
		offset   int64              // 0s
	)

	intervalEnd := IntervalEndsAt(period, offset, time.Unix(now, 0))
	assert.Equal(t, expected, intervalEnd.Unix())
}
