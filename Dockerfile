# Base container image
FROM golang:1.24 AS builder

# Copy code to container
COPY . /src

# Build artifact from /src
WORKDIR /src

# Update the container image and build the binary
RUN go build -ldflags '-s -w' -o leaf cmd/leaf/main.go
######################################################################

# https://github.com/prometheus/prometheus/blob/v3.9.1/Dockerfile
FROM prom/prometheus:v3.9.1 AS prometheus
######################################################################

FROM debian:stable AS runtime
LABEL app=leaf
LABEL org=osba

EXPOSE 9090/tcp
EXPOSE 9010/tcp

RUN mkdir -p /etc/leaf /etc/prometheus

COPY --from=builder /src/entrypoint.sh /.
COPY --from=builder /src/internal/config/config.yaml /etc/leaf/.
COPY --from=builder /src/leaf /bin/.

COPY --from=prometheus /bin/prometheus /bin/.
COPY --from=prometheus /bin/promtool /bin/.
COPY --from=prometheus /etc/prometheus/prometheus.yml /etc/prometheus/.

RUN chmod 755 /bin/leaf

VOLUME [ "/prometheus" ]

ENTRYPOINT [ "/bin/sh" ]
CMD [ "/entrypoint.sh" ]
