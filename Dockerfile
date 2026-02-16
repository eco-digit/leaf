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

COPY --from=builder /src/leaf-bin /.
COPY --from=builder /src/internal/config/config.yaml /.

EXPOSE 9090/tcp
EXPOSE 9091/tcp

# ENTRYPOINT [ "/bin/prometheus" ]
# CMD        [ "--config.file=/etc/prometheus/prometheus.yml", \
             "--storage.tsdb.path=/prometheus" ]

ENTRYPOINT ["/leaf-bin"]
CMD        ["--config", "/config.yaml"]
