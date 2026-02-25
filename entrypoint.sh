#!/bin/sh
/bin/prometheus --config.file=/etc/prometheus/prometheus.yml --storage.tsdb.path=/prometheus &
sleep 3
/bin/leaf --config /etc/leaf/config.yaml
