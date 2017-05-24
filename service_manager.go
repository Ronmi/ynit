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
	"os"
	"path/filepath"
	"strings"
)

// ServiceManager manages services
type ServiceManager struct {
	Services map[string]*Service
	Deps     []string
}

// NewServiceManager creates a ServiceManager instance from a directory
func NewServiceManager(dir string) (ret *ServiceManager, err error) {
	dir = strings.TrimSuffix(dir, "/") + "/"
	ret = &ServiceManager{
		make(map[string]*Service),
		nil,
	}

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil {
			d("Processing %s ...", path)
			if !info.IsDir() {
				d("Adding %s ...", path)
				srv, err := NewService(path)
				if err != nil {
					return err
				}
				ret.Services[path] = srv
			}
		}
		return err
	})

	return
}

// Normalize restructs properties of service.
// It merges StopBefore/StartBefore into StopAfter/StartAfter, and remove non-exist dependencies
func (m *ServiceManager) Normalize() {
	buf := map[string][]*Service{}

	// init buffer
	for _, srv := range m.Services {
		for dep := range srv.Properties[Provides] {
			if _, ok := buf[dep]; !ok {
				buf[dep] = make([]*Service, 0, 1)
			}
			buf[dep] = append(buf[dep], srv)
		}
	}

	// fill deps
	m.Deps = make([]string, 0, len(buf))
	for dep := range buf {
		m.Deps = append(m.Deps, dep)
	}

	// remove non-exist deps
	for _, srv := range m.Services {
		srv.removeNonexist(buf)
	}

	// merge deps
	for _, srv := range m.Services {
		srv.mergeDepend(buf, StartBefore, StartAfter)
		srv.mergeDepend(buf, StopBefore, StopAfter)
	}
}
