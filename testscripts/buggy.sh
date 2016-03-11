#!/bin/bash
### BEGIN INIT INFO
# Provides:       buggy
# Required-Start: 
# Required-Stop:  
# X-Start-Before: nonstop
# X-Stop-After:   nonstop
### END INIT INFO

NAME=bash
EXEC=/bin/bash
PID=/var/run/$NAME.pid

case "$1" in
    start)
        start-stop-daemon --start --oknodo --pidfile $PID --exec $EXEC \
                          --make-pidfile --background -- \
                          /etc/ynit/buggy.sh run
        ;;
    stop)
        start-stop-daemon --stop --oknodo --pidfile $PID --name $NAME
        ;;
    run)
        tail -f /var/log/* /var/log/*/*
        ;;
    *)
        echo "Usage: $0 start|stop"
        exit 1
        ;;
esac
