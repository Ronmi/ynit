#!/bin/bash
### BEGIN INIT INFO
# Provides:       before
# Required-Start: 
# Required-Stop:  
# X-Start-Before: nginx sshd php5-fpm
# X-Stop-After:   nginx sshd php5-fpm
### END INIT INFO

echo "${1}ing before ..."
