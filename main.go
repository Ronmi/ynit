/*
Copyright 2016-2017 Ronmi Ren <ronmi@patrolavia.com>

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
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"golang.org/x/sys/unix"
	syslogd "gopkg.in/mcuadros/go-syslog.v2"
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

type mysyslogd struct {
	server *syslogd.Server
	tcp    string
	udp    string
	unix   string
	format string
}

func (s *mysyslogd) test() bool {
	return s.tcp != "" || s.udp != "" || s.unix != ""
}

func (s *mysyslogd) start() {
	if !s.test() {
		return
	}

	channel := make(syslogd.LogPartsChannel)
	handler := syslogd.NewChannelHandler(channel)

	s.server = syslogd.NewServer()
	switch s.format {
	case "rfc3164":
		s.server.SetFormat(syslogd.RFC3164)
	case "rfc5424":
		s.server.SetFormat(syslogd.RFC5424)
	case "rfc6587":
		s.server.SetFormat(syslogd.RFC6587)
	default:
		s.server.SetFormat(syslogd.Automatic)

	}
	s.server.SetHandler(handler)
	if s.tcp != "" {
		if err := s.server.ListenTCP(s.tcp); err != nil {
			log.Fatalf("Cannot create syslogd at tcp %s: %s", s.tcp, err)
		}
	}
	if s.udp != "" {
		if err := s.server.ListenUDP(s.udp); err != nil {
			log.Fatalf("Cannot create syslogd at udp %s: %s", s.udp, err)
		}
	}
	if s.unix != "" {
		if err := s.server.ListenUnixgram(s.unix); err != nil {
			log.Fatalf("Cannot create syslogd at unix socket path %s: %s", s.unix, err)
		}
	}
	if err := s.server.Boot(); err != nil {
		log.Fatalf("Cannot create syslogd instance: %s", err)
	}

	dp("Syslogd initialized")

	for logParts := range channel {
		var (
			host    string
			content string
			client  string
			t       time.Time
			ok      bool
		)

		host, _ = logParts["hostname"].(string)
		content, _ = logParts["content"].(string)
		client, _ = logParts["client"].(string)
		if t, ok = logParts["timestamp"].(time.Time); !ok {
			t = time.Now()
		}

		fmt.Printf("%s %s %s %s\n", t.Local(), client, host, content)
	}
}

func (s *mysyslogd) stop() {
	if !s.test() {
		return
	}

	_ = s.server.Kill()
}

func main() {
	var (
		confdir        string
		syslogTCPAddr  string
		syslogUDPAddr  string
		syslogUNIXAddr string
		syslogFormat   string
	)
	flag.StringVar(&confdir, "confdir", "/etc/ynit", "Where to read ynit scripts.")
	flag.StringVar(&syslogTCPAddr, "tcp", "", "TCP address:port to listen for buildin tiny syslogd, which is disabled by default.")
	flag.StringVar(&syslogUDPAddr, "udp", "", "UDP address:port to listen for buildin tiny syslogd, which is disabled by default.")
	flag.StringVar(&syslogUNIXAddr, "unix", "", "UNIX socket path to listen for buildin tiny syslogd, which is disabled by default.")
	flag.StringVar(&syslogFormat, "log_format", "RFC3164", "Syslog format, can be rfc3164/rfc5424/rfc6587/auto, only valid if buildin syslogd is enabled.")
	flag.BoolVar(&debug, "debug", false, "Enable debug output")
	flag.Parse()

	logd := &mysyslogd{
		tcp:    syslogTCPAddr,
		udp:    syslogUDPAddr,
		unix:   syslogUNIXAddr,
		format: strings.ToLower(syslogFormat),
	}

	go logd.start()

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
	logd.stop()
	stop(services, processes)
}

func start(services *ServiceManager, processes *ProcessManager) bool {
	e := NewStarter(StartAfter, processes)
	return e.Execute(services)
}

func stop(services *ServiceManager, processes *ProcessManager) {
	e := NewStopper(StopAfter, processes)
	e.Execute(services)
	dp("Service stopped, sending signal to all childs who still alive")
	processes.Find()
	processes.Kill()
	processes.Wait()
}
