DATE := $(shell date +%s)
# DOCKER_TAG := "$(DATE)"
DOCKER_TAG := "latest"
######################################################################

all:
	go build -ldflags '-s -w' -o leaf-bin cmd/leaf/main.go

image:
	docker build . --file Dockerfile --tag "leaf:$(DOCKER_TAG)"
