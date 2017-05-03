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

// Count contains the info about a group of messages
type Count struct {
	Monitor   string
	Unit      string
	Value     uint64
	UUID      string
	Timestamp int64
}

// Alert contains the information abot a message alerting that the
// maximum number of messages has been reached.
type Alert struct {
	Monitor      string `json:"monitor"`
	Type         string `json:"type"`
	UUID         string `json:"uuid,omitempty"`
	CurrentBytes uint64 `json:"current_bytes,omitempty"`
	Limit        uint64 `json:"limit"`
	Timestamp    int64  `json:"timestamp"`
}
