.SILENT:
######################################################################

export DOCKER_DIR := $(CURDIR)/docker
export DOCKER_DEBIAN_RELEASE := stable			# runtime container
export DOCKER_GOLANG_RELEASE := 1.26-alpine		# builder container
export DOCKER_PROMETHEUS_RELEASE := 3.9.1
export DOCKER_TAG := latest
export LEAF_BIN := $(CURDIR)/bin/leaf
export LEAF_CONFIG := $(CURDIR)/internal/config/config.yaml.sample
export LEAF_PROMETHEUS := http://127.0.0.1:9090
export LEAF_PROMETHEUS_PASS := passwd
export LEAF_PROMETHEUS_USER := admin
######################################################################

export LEAF_DOCKER_BUILD_ARGS := \
		--build-arg DOCKER_DEBIAN_RELEASE=$(DOCKER_DEBIAN_RELEASE) \
		--build-arg DOCKER_GOLANG_RELEASE=$(DOCKER_GOLANG_RELEASE) \
		--build-arg DOCKER_PROMETHEUS_RELEASE=$(DOCKER_PROMETHEUS_RELEASE) \
		--build-arg LEAF_PROMETHEUS=$(LEAF_PROMETHEUS) \
		--build-arg LEAF_PROMETHEUS_PASS=$(LEAF_PROMETHEUS_PASS) \
		--build-arg LEAF_PROMETHEUS_USER=$(LEAF_PROMETHEUS_USER)
######################################################################

all:
	go build -ldflags '-s -w' -o $(LEAF_BIN) cmd/leaf/main.go

clean:
	docker image rm -f leaf:$(DOCKER_TAG) 2>/dev/null
	docker image rm -f leaf:$(DOCKER_TAG)_test 2>/dev/null
	rm -f $(LEAF_BIN)

image:
	docker build -q -f $(DOCKER_DIR)/Dockerfile -t leaf:$(DOCKER_TAG) $(LEAF_DOCKER_BUILD_ARGS) .

run: all
	$(LEAF_BIN) --config $(LEAF_CONFIG)

run-image: image
	docker run --rm --name leaf -it \
		-p 9010:9010 \
		-p 9090:9090 \
		leaf:$(DOCKER_TAG)

test:
	docker build -q -f $(DOCKER_DIR)/Dockerfile_test -t leaf:$(DOCKER_TAG)_test $(LEAF_DOCKER_BUILD_ARGS) .
	docker run --rm --name leaf -it \
		-p 9010:9010 \
		-p 9090:9090 \
		leaf:$(DOCKER_TAG)_test
