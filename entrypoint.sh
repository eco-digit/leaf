#!/bin/sh

if [ -f /bin/prometheus ]; then
    /bin/prometheus --config.file=/etc/prometheus/prometheus.yml --storage.tsdb.path=/prometheus &
fi
sleep 3

/bin/leaf --config /etc/leaf/config.yaml
