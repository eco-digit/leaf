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

## Local Deployment for testing while working on the codebase
Connect to prometheus via port forwarding:
1. Start port forwarding from local host to a running prometheus server:
`ssh -i ~/.ssh/id_ed25519 -L 9091::9091 user@server`

2. Start leaf, setting the config path flag:
`go run ./cmd/leaf --config config/config.yaml.sample`

3. Check /metrics at http://localhost:9010/metrics for embodied impact metrics

4. Test Collector: `go run ./cmd/leaf -config config/config.yaml.sample -collect-once`
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

  * **Environment Variables**

    Here's a list of the current supported variables, and thei respective default values:

    - DOCKER_DEBIAN_RELEASE := stable
    - DOCKER_GOLANG_RELEASE := 1.26-alpine
    - DOCKER_PROMETHEUS_RELEASE := 3.9.1
    - LEAF_DOCKER_TAG := latest
    - LEAF_PROMETHEUS := http://127.0.0.1:9090
    - LEAF_PROMETHEUS_PASS := passwd
    - LEAF_PROMETHEUS_USER := admin

> Should you be willing to change some of the values please run `make image VARIABLE=value`

### Starting exporter manually

  * **Running the binary locally**

    ```sh
    make run
    ```

  * **Running the binary in a Docker container**

    ```sh
    make run-image
    ```

### Test 'leaf' in a container with Prometheus

This depends on the `make image` target (but this dependency is sorted via `Makefile` already). A new image will be built with the `_test` suffix and it will be used to start a Prometheus instance directly on a container with `leaf`.

```sh
make test
```

> Once all runs fine, the environment variables will be printed out.

### Cleaning binary artifact and Docker images

```sh
make clean
```
