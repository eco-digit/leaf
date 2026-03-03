.SILENT:
######################################################################

DOCKER_DIR := $(CURDIR)/docker
DOCKER_GOLANG_RELEASE := 1.24
DOCKER_PROMETHEUS_RELASE = 3.9.1
DOCKER_TAG := latest
LEAF_BIN := $(CURDIR)/bin/leaf
LEAF_CONFIG := $(CURDIR)/internal/config/config.yaml
LEAF_PROMETHEUS := 127.0.0.1
LEAF_PROMETHEUS_PASS := passwd
LEAF_PROMETHEUS_USER := admin
######################################################################

export DOCKER_GOLANG_RELEASE
export DOCKER_PROMETHEUS_RELASE
######################################################################

all:
	go build -ldflags '-s -w' -o $(LEAF_BIN) cmd/leaf/main.go

clean:
	docker image rm -f leaf:$(DOCKER_TAG) 2>/dev/null
	docker image rm -f leaf:$(DOCKER_TAG)_test 2>/dev/null
	rm -f $(LEAF_BIN)

image:
	docker build -q -f $(DOCKER_DIR)/Dockerfile -t leaf:$(DOCKER_TAG) .

run: all
	$(LEAF_BIN) --config $(LEAF_CONFIG)

run-image: image
	docker run --rm --name leaf -it \
		-p 9010:9010 \
		-p 9090:9090 \
		leaf:$(DOCKER_TAG)

test:
	docker build -q -f $(DOCKER_DIR)/Dockerfile_test -t leaf:$(DOCKER_TAG)_test \
		--build-arg DOCKER_GOLANG_RELEASE=$(DOCKER_GOLANG_RELEASE) \
		--build-arg DOCKER_PROMETHEUS_RELEASE=$(DOCKER_PROMETHEUS_RELEASE) \
		--build-arg LEAF_PROMETHEUS=$(LEAF_PROMETHEUS) \
		--build-arg LEAF_PROMETHEUS_PASS=$(LEAF_PROMETHEUS_PASS) \
		--build-arg LEAF_PROMETHEUS_USER=$(LEAF_PROMETHEUS_USER) .
