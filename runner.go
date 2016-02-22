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
	"os"
	"os/exec"
)

func runner(srv *service, arg string, notify chan string) {
	script := srv.script
	depOld := srv.startAfter
	if arg != "start" {
		depOld = srv.stopAfter
	}
	deps := make([]string, 0, len(depOld))
	for dep, ok := range depOld {
		if ok {
			deps = append(deps, dep)
		}
	}

	done := make(chan int)

	go func(notify chan string, done chan int) {
		// service runner
		flags := map[string]bool{}
		if len(deps) > 0 {
			for x := range notify {
				flags[x] = true
				fulfill := true

				for _, dep := range deps {
					if !flags[dep] {
						fulfill = false
						break
					}
				}

				if fulfill {
					break
				}
			}
		}

		d("Script %s is %sing", srv.script, arg)
		go func(script, arg string) {
			cmd := exec.Command(script, arg)
			cmd.Stdout = os.Stderr // redirect to stderr so you can see it in docker logs
			cmd.Stderr = os.Stderr
			_ = cmd.Run()
			done <- 1
		}(script, arg)

		for range notify {
		}
		d("runner for %s stopped", script)
	}(notify, done)

	<-done
}
