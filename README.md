# LEAF

LEAF stands for '**L**ifecycle-based **E**nvironmental **A**ssessment of **F**ootprints' :leaves:

Currently, this exporter/service is only running locally. You may connect it to a prometheus server using port forwarding, or changing the configuration file accordingly. 

Before proceeding any further, ensure you have the necessary softwares to use the code served on this repository.

  - Docker
  - Go (golang)
  - make
  - Prometheus

> NOTE: leaf requires that prometheus is available and running, otherwise it will be terminated and constantly restarted until it connects to the service.

Here'a a tree of files present so far:

```
leaf/
├── cmd/
│   └── leaf/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── promclient/
│   │   └── client.go
│   └── exporter/
│       └── exporter.go
├── go.mod
├── go.sum
└── config.yaml
```

### Building the binary

```sh
make
```

### Building container image

```sh
make image
```

### Starting exporter manually

```shell
go run ./cmd/leaf --config internal/config/config.yaml
```
