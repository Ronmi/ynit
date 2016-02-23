/*
Copyright 2016 Ronmi Ren <ronmi@patrolavia.com>

This file is part of YNIT.

YNIT is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

YNIT is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with YNIT.  If not, see <http://www.gnu.org/licenses/>.
*/

package main

import (
	"flag"
	"log"
	"os"
	"os/signal"

	"golang.org/x/sys/unix"
)

var debug bool

func d(fmt string, vars ...interface{}) {
	if debug {
		log.Printf(fmt, vars...)
	}
}

func dp(vars ...interface{}) {
	if debug {
		log.Print(vars...)
	}
}

func main() {
	var confdir string
	flag.StringVar(&confdir, "confdir", "/etc/ynit", "where to read ynit scripts")
	flag.BoolVar(&debug, "debug", false, "Enable debug output")
	flag.Parse()

	services, err := NewServiceManager(confdir)
	if err != nil {
		log.Fatalf("Error parsing %s: %s", confdir, err)
	}
	services.Normalize()
	processes := NewPM()

	if !start(services, processes) {
		log.Print("Cannot start all services, quitting.")
		stop(services, processes)
		log.Fatal("Quitting")
	}
	dp("Service started, waiting for child processes")

	term := make(chan os.Signal, 1)
	signal.Notify(term, unix.SIGTERM, unix.SIGINT)

	<-term
	stop(services, processes)
	dp("Service stopped, waiting for child processes")
	processes.Find()
	processes.Wait()
}

func start(services *ServiceManager, processes *ProcessManager) bool {
	e := NewExecutor("start", StartAfter, processes)
	return e.Execute(services)
}

func stop(services *ServiceManager, processes *ProcessManager) bool {
	e := NewExecutor("stop", StopAfter, processes)
	return e.Execute(services)
}
