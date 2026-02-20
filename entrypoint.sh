#!/bin/sh

/bin/prometheus --config.file=/etc/prometheus/prometheus.yml --storage.tsdb.path=/prometheus &

/leaf-bin --config /config.yaml &
