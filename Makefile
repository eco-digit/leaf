.SILENT:
######################################################################

DOCKER_DIR := $(CURDIR)/docker
DOCKER_TAG := latest
LEAF_BIN := $(CURDIR)/bin/leaf
LEAF_CONFIG := $(CURDIR)/internal/config/config.yaml
LEAF_PROM_PASS := passwd
LEAF_PROM_USER := admin
######################################################################

all:
	go build -ldflags '-s -w' -o $(LEAF_BIN) cmd/leaf/main.go

clean:
	docker image rm -f leaf:$(DOCKER_TAG) 2>/dev/null
	rm -f $(LEAF_BIN)

image:
	docker build -f $(DOCKER_DIR)/Dockerfile -t leaf:$(DOCKER_TAG) .

run: all
	# go run ./cmd/leaf --config $(LEAF_CONFIG)
	$(LEAF_BIN) --config $(LEAF_CONFIG)
