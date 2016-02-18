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
	"bufio"
	"os"
	"strings"
)

func setProp(line, prop string, data map[string]bool) {
	str := strings.TrimSpace(strings.TrimPrefix(line, prop))
	if str == "" {
		return
	}
	items := strings.Split(str, " ")
	for _, item := range items {
		if item == "" {
			continue
		}
		data[item] = true
	}
}

type service struct {
	provides    map[string]bool // Provides
	startAfter  map[string]bool // Required-Start
	stopBefore  map[string]bool // Required-Stop
	startBefore map[string]bool // X-Start-Before
	stopAfter   map[string]bool // X-Stop-After
	script      string
}

type services struct {
	data map[string]*service
	srvs map[string][]*service
}

func newServices() *services {
	return &services{
		map[string]*service{},
		map[string][]*service{},
	}
}

func (s *services) load(script string) (err error) {
	f, err := os.Open(script)
	if err != nil {
		return
	}
	defer f.Close()

	begin := false
	srv := &service{
		map[string]bool{},
		map[string]bool{},
		map[string]bool{},
		map[string]bool{},
		map[string]bool{},
		script,
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !begin {
			if strings.TrimRight(line, " \t\r\n") == "### BEGIN INIT INFO" {
				begin = true
			}
			continue
		}

		if strings.TrimRight(line, " \t\r\n") == "### END INIT INFO" {
			break
		}

		if !strings.HasPrefix(line, "# ") {
			continue
		}

		switch {
		case strings.HasPrefix(line, "# Provides:"):
			setProp(line, "# Provides:", srv.provides)
		case strings.HasPrefix(line, "# Required-Start:"):
			setProp(line, "# Required-Start:", srv.startAfter)
		case strings.HasPrefix(line, "# Required-Stop:"):
			setProp(line, "# Required-Stop:", srv.stopBefore)
		case strings.HasPrefix(line, "# X-Start-Before:"):
			setProp(line, "# X-Start-Before:", srv.startBefore)
		case strings.HasPrefix(line, "# X-Stop-After:"):
			setProp(line, "# X-Stop-After:", srv.stopAfter)
		}

	}

	if _, ok := s.data[script]; !ok {
		s.data[script] = srv
		for p := range srv.provides {
			if _, ok := s.srvs[p]; !ok {
				s.srvs[p] = make([]*service, 0, 1)
			}
			s.srvs[p] = append(s.srvs[p], srv)
		}
	}

	return scanner.Err()
}

// normalize removes non-exist dependencies, and merge startBefore/stopBefore to startAfter/stopAfter
func (s *services) normalize() {
	// first loop, remove non-exist deps
	for _, srv := range s.data {
		s.filterDeps(srv.provides)
		s.filterDeps(srv.startAfter)
		s.filterDeps(srv.stopBefore)
		s.filterDeps(srv.startBefore)
		s.filterDeps(srv.stopAfter)
	}

	// second loop, merge deps
	for _, srv := range s.data {
		s.mergeStart(srv)
		s.mergeStop(srv)
	}
}

func (s *services) filterDeps(data map[string]bool) {
	for srv := range data {
		if _, ok := s.srvs[srv]; !ok {
			data[srv] = false
		}
	}
}

func (s *services) mergeStart(srv *service) {
	for dep, ok := range srv.startBefore {
		if !ok {
			continue
		}
		for _, dest := range s.srvs[dep] {
			for provide := range srv.provides {
				dest.startAfter[provide] = true
			}
		}
	}
}

func (s *services) mergeStop(srv *service) {
	for dep, ok := range srv.stopBefore {
		if !ok {
			continue
		}
		for _, dest := range s.srvs[dep] {
			for provide := range srv.provides {
				dest.stopAfter[provide] = true
			}
		}
	}
}

func (s *services) start() {
	s.run("start")
}

func (s *services) stop() {
	s.run("stop")
}

func (s *services) run(arg string) {
	chs := make(map[string]chan string)
	wait := make(chan int)
	done := make(chan int)
	for _, srv := range s.data {
		go func(chs map[string]chan string, srv *service) {
			ch := make(chan string)
			chs[srv.script] = ch
			runner(srv, arg, ch)
			for range wait {
			}
			for _, c := range chs {
				for dep := range srv.provides {
					c <- dep
				}
			}
			done <- 1
		}(chs, srv)
	}
	close(wait)
	for range s.data {
		<-done
	}
}
