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
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/sys/unix"
)

// PROC denotes procfs root
const PROC = "/proc"

// ProcessManager manages adopted processes
type ProcessManager struct {
	*sync.Mutex
	monitoring map[int]bool
	*sync.WaitGroup
}

// NewPM creates new ProcessManager instance
func NewPM() *ProcessManager {
	ret := &ProcessManager{
		&sync.Mutex{},
		map[int]bool{},
		&sync.WaitGroup{},
	}

	go func(p *ProcessManager) {
		chld := make(chan os.Signal, 1)
		signal.Notify(chld, unix.SIGCHLD)
		for range chld {
			p.Find()
		}
	}(ret)
	return ret
}

// Run a command in subprocess without adopting it again
func (m *ProcessManager) Run(script, arg string) (err error) {
	cmd := exec.Command(script, arg)
	cmd.Stdout = os.Stderr // redirect to stderr so you can see it in docker logs
	cmd.Stderr = os.Stderr
	m.Lock()
	defer m.Unlock()
	if err = cmd.Start(); err != nil {
		return
	}
	pid := cmd.Process.Pid
	m.monitoring[pid] = true
	m.Unlock()
	err = cmd.Wait()
	m.Lock()
	m.monitoring[pid] = false
	return
}

// Find out adopted processes
func (m *ProcessManager) Find() {
	m.Lock()
	defer m.Unlock()
	myid := os.Getpid()
	fis, err := ioutil.ReadDir(PROC)
	if err != nil {
		return
	}

	for _, fi := range fis {
		if !fi.IsDir() {
			continue
		}

		m.adopt(myid, fi.Name())
	}
}

// check if it is child process, adopt if yes and yet adopted
func (m *ProcessManager) adopt(myid int, name string) {
	pid, err := strconv.Atoi(name)
	if err != nil || pid <= 1 {
		// not child process
		return
	}

	if m.monitoring[pid] {
		// already monitoring
		return
	}

	if !m.isChild(myid, name) {
		// not child process
		return
	}

	p, err := os.FindProcess(pid)
	if err != nil {
		return
	}

	m.monitoring[pid] = true
	m.Add(1)
	go m.reap(p, pid)

	// fetch cmdline
	cmd := ""
	if cmds, err := ioutil.ReadFile(PROC + "/" + name + "/cmdline"); err == nil {
		cmd = strings.Replace(string(cmds), "\x00", " ", -1)
	}
	d("Monitoring child %d %s", pid, cmd)
}

// reap a child process
func (m *ProcessManager) reap(p *os.Process, pid int) {
	_, _ = p.Wait()
	m.Lock()
	m.monitoring[pid] = false
	m.Unlock()
	m.Done()
	d("Child process %d exited", pid)
}

// isChild parses /proc/*pid*/status to find and compare ppid
func (m *ProcessManager) isChild(myid int, pid string) bool {
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
