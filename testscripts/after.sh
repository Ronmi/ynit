#!/bin/bash
### BEGIN INIT INFO
# Provides:       after
# Required-Start: nginx sshd php5-fpm
# Required-Stop:  nginx sshd php5-fpm
# X-Start-Before:
# X-Stop-After:
### END INIT INFO

echo "${1}ing after ..."
