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
	"strconv"
	"strings"
	"sync"
)

const PROC = "/proc"

type childMgr struct {
	monitoring map[int]bool
	*sync.Mutex
	*sync.WaitGroup
}

func newMgr() *childMgr {
	return &childMgr{
		map[int]bool{},
		&sync.Mutex{},
		&sync.WaitGroup{},
	}
}

// find out adopted processes
func (m *childMgr) adopt() {
	myid := os.Getpid()
	f, err := os.Open(PROC)
	if err != nil {
		return
	}
	defer f.Close()

	fis, err := f.Readdir(0)
	if err != nil {
		return
	}

	for _, fi := range fis {
		if !fi.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(fi.Name())
		if err != nil || pid <= 1 {
			// not child process
			continue
		}

		if m.monitoring[pid] {
			// already monitoring
			continue
		}

		if m.isChild(myid, fi.Name()) {
			p, err := os.FindProcess(pid)
			if err != nil {
				continue
			}
			m.Lock()
			m.monitoring[pid] = true
			m.Add(1)
			go m.reap(p, pid)
			m.Unlock()
			d("Monitoring child %d", pid)
		}
	}
}

// reap a child process
func (m *childMgr) reap(p *os.Process, pid int) {
	p.Wait()
	m.Lock()
	m.monitoring[pid] = false
	m.Unlock()
	m.Done()
	d("Child process %d exited", pid)
}

// isChild parses /proc/*pid*/status to find and compare ppid
func (m *childMgr) isChild(myid int, pid string) bool {
	ppid := strconv.Itoa(myid)

	f, err := os.Open(PROC + "/" + pid + "/status")
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.ToLower(scanner.Text())
		if !strings.HasPrefix(line, "ppid:") {
			continue
		}

		actual := strings.TrimSpace(string(line[5:]))
		return actual == ppid
	}
	return false
}
