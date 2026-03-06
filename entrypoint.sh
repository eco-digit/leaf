#!/bin/sh

if [ -f /bin/prometheus ]; then
    /bin/prometheus --config.file=/etc/prometheus/prometheus.yml --storage.tsdb.path=/prometheus &
fi
sleep 3

if [ -n $LEAF_DOCKER_ENV ]; then
    /bin/leaf --config /etc/leaf/config.yaml
else
    /bin/leaf --config /etc/leaf/config.yaml &
    sleep 7
    env
    kill -s SIGKILL 1
fi
