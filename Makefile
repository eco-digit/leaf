.SILENT:
######################################################################

DOCKER_DIR := $(CURDIR)/docker
DOCKER_TAG := latest
LEAF_BIN := $(CURDIR)/bin/leaf
LEAF_CONFIG := $(CURDIR)/internal/config/config.yaml
LEAF_PROMETHEUS := 127.0.0.1
LEAF_PROMETHEUS_PASS := passwd
LEAF_PROMETHEUS_USER := admin
######################################################################

all:
	go build -ldflags '-s -w' -o $(LEAF_BIN) cmd/leaf/main.go

clean:
	docker image rm -f leaf:$(DOCKER_TAG) 2>/dev/null
	rm -f $(LEAF_BIN)

image:
	docker build -q -f $(DOCKER_DIR)/Dockerfile -t leaf:$(DOCKER_TAG) \
		--build-arg LEAF_PROMETHEUS=$(LEAF_PROMETHEUS) \
		--build-arg LEAF_PROMETHEUS_PASS=$(LEAF_PROMETHEUS_PASS) \
		--build-arg LEAF_PROMETHEUS_USER=$(LEAF_PROMETHEUS_USER) \
		.

run: all
	# go run ./cmd/leaf --config $(LEAF_CONFIG)
	$(LEAF_BIN) --config $(LEAF_CONFIG)

run-image: image
	docker run --rm --name leaf -it leaf:$(DOCKER_TAG)
