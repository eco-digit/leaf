DATE := $(shell date +%s)
# DOCKER_TAG := "$(DATE)"
DOCKER_TAG := latest
######################################################################

all:
	go build -ldflags '-s -w' -o leaf-bin cmd/leaf/main.go

image:
	docker build -f Dockerfile -t leaf:$(DOCKER_TAG) .

test:
	go run ./cmd/leaf --config internal/config/config.yaml
