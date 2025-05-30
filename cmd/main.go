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

import (
	"flag"
	"io/ioutil"
	"os"
	"os/signal"
	"sync"

	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

var (
	version    string
	configFile string

	wg        = new(sync.WaitGroup)
	terminate = make(chan struct{})
)

var log = logrus.New()

func init() {
	log.Formatter = &prefixed.TextFormatter{
		ForceColors:      true,
		DisableTimestamp: true,
	}

	versionFlag := flag.Bool("version", false, "Show version info")
	debugFlag := flag.Bool("debug", false, "Show debug info")
	configFlag := flag.String("config", "", "Application configuration file")
	flag.Parse()

	if *versionFlag {
		PrintVersion()
		os.Exit(0)
	}

	if len(*configFlag) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	if *debugFlag {
		log.Level = logrus.DebugLevel
	}

	configFile = *configFlag
}

func main() {
	///////////////////////
	// App Configuration //
	///////////////////////

	rawConfig, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalln("Error reading config file: " + err.Error())
	}

	config, err := ParseConfig(rawConfig)
	if err != nil {
		log.Fatalln("Error parsing config: " + err.Error())
	}

	/////////////////////////////////////////////////
	// Start the pipeline for accounting messages //
	/////////////////////////////////////////////////

	UUIDCountersPipeline(config)

	//////////////////////////////////////////////
	// Start the pipeline for limits reporting //
	//////////////////////////////////////////////

	CountersMonitor(config)

	///////////////////
	// Handle SIGINT //
	///////////////////

	sigint := make(chan os.Signal)
	signal.Notify(sigint, os.Interrupt)

	<-sigint
	close(terminate)

	/////////////
	// The End //
	/////////////

	wg.Wait()
	log.Infoln("Bye bye...")
}
