# LEAF

LEAF stands for '**L**ifecycle-based **E**nvironmental **A**ssessment of **F**ootprints' :leaves:

Currently, this exporter/service is only running locally. You may connect it to a prometheus server using port forwarding (or changing the configuration file accordingly). 

Before proceeding any further, ensure you have the necessary softwares to use the code served on this repository.

  - Docker
  - Go (golang)
  - make
  - Prometheus

> NOTE: `leaf` requires that prometheus is available and running, otherwise it will be terminated and constantly restarted until it connects to the service. Create a port forwarding (perhaps using a `ssh` tunnel), as mentioned above, or change the configuration file to access an instance of prometheus before starting `leaf`.

Here'a a `tree` of files present so far:

```
.
├── bin
├── cmd
│   └── leaf
│       └── main.go
├── docker
│   ├── Dockerfile
│   └── Dockerfile_test
├── entrypoint.sh
├── go.mod
├── go.sum
├── internal
│   ├── config
│   │   ├── config.go
│   │   └── config.yaml
│   ├── exporter
│   │   └── exporter.go
│   ├── model
│   │   └── leaf-model.yaml
│   └── promclient
│       └── client.go
├── LICENSE
├── Makefile
└── README.md
```

### Building the binary

```sh
make
```

### Building a Docker container image

```sh
make image
```

### Starting exporter manually

  * **Running the binary locally**

```sh
make run
```

  * **Running the binary in a Docker container*

```sh
make run-image
```
