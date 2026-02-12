# Leaf 

## **L**ifecycle based **E**nvironmental **A**ssessment of **F**ootprints

Currently, this exporter/service is only running locally you can connect it to a prometheus server using port forwarding. 

Before proceedin any further, ensure you have the necessary softwares to use the code served on this repository.

  - Docker
  - Go (golang)
  - make
  - Prometheus

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

### Building the binary:

```sh
make
```

### Building the container image:

```sh
make image
```

### Running on the fly:

# Start exporter
```shell
go run ./cmd/leaf --config config.yaml
```
