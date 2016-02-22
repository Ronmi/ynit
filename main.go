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
	"path/filepath"
	"strings"

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

	confdir = strings.TrimSuffix(confdir, "/") + "/"
	data := newServices()
	mgr := newMgr()

	filepath.Walk(confdir, func(path string, info os.FileInfo, err error) error {
		if err == nil {
			d("Processing %s ...", path)
			if !info.IsDir() {
				d("Adding %s ...", path)
				data.load(path)
			}
		}
		return err
	})

	data.normalize()
	data.start(mgr)
	dp("Service started")

	go func() {
		chld := make(chan os.Signal, 1)
		signal.Notify(chld, unix.SIGCHLD)
		for range chld {
			mgr.adopt()
		}
	}()

	term := make(chan os.Signal, 1)
	signal.Notify(term, unix.SIGTERM, unix.SIGINT)

	<-term
	data.stop(mgr)
	dp("Service stopped, waiting for child processes")
	mgr.adopt()
	mgr.Wait()

}
