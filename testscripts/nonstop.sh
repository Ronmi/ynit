#!/bin/bash
### BEGIN INIT INFO
# Provides:       nonstop
# Required-Start: nginx sshd php5-fpm
# Required-Stop:  nginx sshd php5-fpm
# X-Start-Before:
# X-Stop-After:
# Non-Stop: true
### END INIT INFO

tail -F /var/log/nginx/*
