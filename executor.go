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

import "sync"

// ExecuteResult represents result of ynit script execution
type ExecuteResult struct {
	Service *Service
	Result  State // must be one of Success of Failed
}

// Executor executes all ynit script
type Executor struct {
	arg           string   // arg pass to ynit script
	prop          Property // parse deps using this property, must be one of StartAfter or StopAfter
	pm            *ProcessManager
	serviceStates map[*Service]State
	depStates     map[string]State
	result        chan *ExecuteResult
	*sync.Mutex
	done chan bool
}

// NewExecutor creates an executor
func NewExecutor(arg string, prop Property, pm *ProcessManager) *Executor {
	return &Executor{
		arg,
		prop,
		pm,
		map[*Service]State{},
		map[string]State{},
		make(chan *ExecuteResult, 1),
		new(sync.Mutex),
		make(chan bool),
	}
}

func (e *Executor) exec(srv *Service) {
	ret := &ExecuteResult{
		srv,
		Success,
	}
	if err := e.pm.Run(srv.Script, e.arg); err != nil {
		ret.Result = Failed
	}
	e.result <- ret
}

// Execute ynit script
func (e *Executor) Execute(m *ServiceManager) bool {
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

	return <-e.done
}

func (e *Executor) trigger() {
	for result := range e.result {
		e.Lock()
		e.serviceStates[result.Service] = result.Result
		if len(e.result) == 0 {
			e.parse()
		}
		e.Unlock()
	}
}

func (e *Executor) parse() {
	haveRunnable := false
	haveError := false

	for srv, state := range e.serviceStates {
		// set flags
		switch state {
		case Pending, Waiting, Running:
			haveRunnable = true
		case Failed, Error:
			haveError = true
		}

		if state == Pending {
			state = srv.Can(e.depStates, e.prop)
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
		e.done <- haveError
	}
}
