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

// State of service
type State string

// possible states
const (
	Error   State = "error" // one (or more) dependencies is in failed state
	Pending State = "pending"
	Waiting State = "waiting"
	Running State = "running"
	Success State = "success"
	Failed  State = "failed"
)

// Property of service
type Property string

// possible properties
const (
	Provides    Property = "# Provides:"
	StartAfter  Property = "# Required-Start:"
	StopBefore  Property = "# Required-Stop:"
	StartBefore Property = "# X-Start-Before:"
	StopAfter   Property = "# X-Stop-After:"
)

// all properties
var (
	Props = []Property{
		Provides, // this will always lay on element 0
		StartAfter,
		StopBefore,
		StartBefore,
		StopAfter,
	}
)

// Service info
type Service struct {
	Properties map[Property]map[string]bool
	Script     string
}

// NewService creates a Service instance by parsing script
func NewService(script string) (ret *Service, err error) {
	f, err := os.Open(script)
	if err != nil {
		return
	}
	defer f.Close()

	props := make(map[Property]map[string]bool)
	for _, prop := range Props {
		props[prop] = make(map[string]bool)
	}
	props[Provides][script] = true // must provide script itself

	ret = &Service{
		props,
		script,
	}

	scanner := bufio.NewScanner(f)
	begin := false

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

		for _, prop := range Props {
			if strings.HasPrefix(line, string(prop)) {
				ret.setProp(line, prop)
				break
			}
		}
	}

	return
}

func (s *Service) setProp(line string, prop Property) {
	str := strings.TrimSpace(strings.TrimPrefix(line, string(prop)))
	if str == "" {
		return
	}
	items := strings.Split(str, " ")
	for _, item := range items {
		if item == "" {
			continue
		}
		s.Properties[prop][item] = true
	}
}

// Can detects if all dependencies of the Service is fulfilled
func (s *Service) Can(state map[string]State, prop Property) State {
	for dep := range s.Properties[prop] {
		if state[dep] == Failed {
			return Error
		}
		if state[dep] != Success {
			return Pending

		}
	}
	return Waiting
}

func (s *Service) removeNonexist(buf map[string][]*Service) {
	props := Props[1:]
	for _, prop := range props {
		for dep := range s.Properties[prop] {
			if _, ok := buf[dep]; !ok {
				delete(s.Properties[prop], dep)
			}
		}
	}
}

func (s *Service) mergeDepend(buf map[string][]*Service, from, to Property) {
	for want := range s.Properties[from] {
		for _, victim := range buf[want] {
			victim.Properties[to][s.Script] = true
		}
	}

	s.Properties[from] = map[string]bool{}
}
