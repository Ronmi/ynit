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
	"sync"
	"syscall"
)

// Stopper executes all ynit script
type Stopper struct {
	prop          Property // parse deps using this property, must be one of StartAfter or StopAfter
	pm            *ProcessManager
	serviceStates map[*Service]State
	depStates     map[string]State
	result        chan *ExecuteResult
	*sync.Mutex
	done chan bool
}

// NewStopper creates an executor
func NewStopper(prop Property, pm *ProcessManager) *Stopper {
	return &Stopper{
		prop,
		pm,
		map[*Service]State{},
		map[string]State{},
		make(chan *ExecuteResult, 1),
		new(sync.Mutex),
		make(chan bool),
	}
}

func (e *Stopper) exec(srv *Service) {
	d("Stopping %s ...", srv.Script)
	ret := &ExecuteResult{
		srv,
		Success,
	}

	if srv.IsNonStop() {
		ret.Result = Failed
		if srv.Process != nil {
			if err := srv.Process.Signal(syscall.SIGINT); err == nil {
				ret.Result = Success
			}
		}
	} else {
		if err := e.pm.Run(srv.Script, "stop"); err != nil {
			ret.Result = Failed
		}
	}
	d("Result of %s stop: %s", srv.Script, ret.Result)
	e.result <- ret
}

// Execute ynit script
func (e *Stopper) Execute(m *ServiceManager) bool {
	// initialize states
	e.Lock()
	for _, srv := range m.Services {
		e.serviceStates[srv] = Pending
	}
	for _, dep := range m.Deps {
		e.depStates[dep] = Pending
	}
	e.Unlock()

	go e.trigger()
	e.Lock()
	e.parse()
	e.Unlock()

	ret := <-e.done
	return ret
}

func (e *Stopper) trigger() {
	for result := range e.result {
		e.Lock()
		e.serviceStates[result.Service] = result.Result

		for dep := range result.Service.Properties[Provides] {
			if result.Result == Success {
				e.depStates[dep] = Success
				continue
			}

			if e.depStates[dep] != Success {
				e.depStates[dep] = result.Result
			}
		}

		if len(e.result) == 0 {
			e.parse()
		}
		e.Unlock()
	}
}

func (e *Stopper) parse() {
	haveRunnable := false

	for srv, state := range e.serviceStates {
		// set flags
		switch state {
		case Pending, Waiting, Running:
			haveRunnable = true
		}

		if state == Pending {
			state = srv.CanStop(e.depStates, e.prop)
			e.serviceStates[srv] = state
		}

		if state == Waiting {
			e.serviceStates[srv] = Running
			haveRunnable = true
			go e.exec(srv)
			continue
		}
	}

	if !haveRunnable {
		close(e.result)
		e.done <- false
	}
}
