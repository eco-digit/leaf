# Base container image
FROM golang:1.24 AS builder

# Copy code to container
COPY . /src

# Build artifact from /src
WORKDIR /src

# Update the container image and build the binary
RUN go build -ldflags '-s -w' -o leaf-bin cmd/leaf/main.go
######################################################################

FROM prom/prometheus:main AS runtime

# EXPOSE 9090/tcp
# EXPOSE 9091/tcp

COPY entrypoint.sh /entrypoint.sh

COPY --from=builder /src/leaf-bin /leaf-bin
COPY --from=builder /src/internal/config/config.yaml /config.yaml

# ENTRYPOINT [ "/bin/prometheus" ]
# CMD        [ "--config.file=/etc/prometheus/prometheus.yml", "--storage.tsdb.path=/prometheus" ]
#
# ENTRYPOINT [ "/leaf-bin" ]
# CMD        [ "--config", "/config.yaml" ]
#
# ENTRYPOINT [ "/bin/sh", "-c" ]
# CMD        [ "/bin/prometheus --config.file=/etc/prometheus/prometheus.yml --storage.tsdb.path=/prometheus", "\&\;", "echo HA" ]

ENTRYPOINT [ "/bin/sh", "-c", "entrypoint.sh" ]
