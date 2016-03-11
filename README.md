# YNIT - tiny program to handle PID 1 problem for docker container

YNIT is `init` process supports sysv-like format scripts. I aims to handle PID 1 problems in docker.

[![Go Report Card](https://goreportcard.com/badge/github.com/Ronmi/ynit)](https://goreportcard.com/report/github.com/Ronmi/ynit)

#### PID 1 problem?

See [phusion's awesome blog post](https://blog.phusion.nl/2015/01/20/docker-and-the-pid-1-zombie-reaping-problem/) for detail.

#### Why recreate the wheel

Because golang is cool!

## Script format

YNIT script is just a shell script accepts only one parameter: `start` or `stop`. You can set dependencies by adding some properties in special format, which is roughly like the scripts in `/etc/init.d/`.

A typical YNIT script will be like:

```sh
#!/bin/bash
### BEGIN INIT INFO
# Provides:       myprog mytest-prog
# Required-Start: another-script
# Required-Stop:
# X-Start-Before:
# X-Stop-After:
# Non-Stop:
### END INIT INFO

PIDFILE=/tmp/myprog.pid

case "$1" in
  start)
    start-stop-daemon --start --pidfile $PIDFILE --exec /usr/bin/myprog -- --daemon 
    ;;
  stop)
    start-stop-daemon --stop --oknodo --pidfile $PIDFILE
    ;;
  *)
    echo "Usage: $0 start|stop" >&2
    exit 1
    ;;
esac
```

or

```php
#!/usr/bin/env php
<?php
/*
### BEGIN INIT INFO
# Provides:       myprog mytest-prog
# Required-Start: another-script
# Required-Stop:
# X-Start-Before:
# X-Stop-After:
# Non-Stop:
### END INIT INFO
*/

// here are codes to handle command line arguments and start/stop your program
```

As you can see, it is almost compatible with the scripts in `/etc/init.d/`. In fact, you can just make a symlink to YNIT directory instead of writing your own script.

## Non-stop jobs

You can also run a program without forking it to background. This can be done easily by setting `Non-Stop` property to `yes` or `true.

```sh
#!/bin/bash
### BEGIN INIT INFO
# Provides:       log-redirector
# Required-Start: service1 service2 and all services
# Required-Stop:  service1 service2 and all services
# X-Start-Before:
# X-Stop-After:
# Non-Stop:
### END INIT INFO

tail -f /log/of/service1 /log/of/other/services
```

This example is an useful trick to forward service logs to docker.

#### WARNING
- Property lines MUST begin with `# `, no other characters allowed.
- Property block is case-sensitive.
- The `### BEGIN INIT INFO` and `### END INIT INFO` lines are required, and must begin with `### `.
- Only first property block is parsed.
- No variable subsitution.

## How it works

YNIT reads all scripts in `/etc/ynit/`, parse for properties (if exist), and executes them asynchronously. To be compatible with scripts in `/etc/init.d/`, dependency will be ignored if not exists.

It will create separated runner in goroutines for each script. The runner waits for dependencies to finish if there are some, and broadcasts its name to other runners when itself finished.

After services are started, YNIT sleeps in background, waiting for `SIGTERM` or `SIGINT` to stop services.

## How to test it

Since YNIT is mainly build for running in docker container, you will need a running docker environment to test it.

1. Build ynit binary with `go build`.
2. Build test image with `docker build -t ynit .`
3. Run a container with `docker run --rm --name ynit`.
4. Exec' into the container with `docker exec -it ynit ps ax`, see if ssh, nginx and php-fpm services are all up.
5. Stop the container with `docker stop ynit`
6. Extract log files from container and examine if they were gracefully exited.

## License

GNU General Public License
