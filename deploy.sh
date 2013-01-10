#!/bin/sh

set -e

glp build
scp pratbot pratbot.ctrl-c.us:~/servers/pratbot
rm pratbot
