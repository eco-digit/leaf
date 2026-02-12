DATE := $(shell date +%s)
DOCKER_TAG := "leaf:$(DATE)"
######################################################################

all:
	go build -ldflags '-s -w' -o leaf-bin cmd/leaf/main.go

image:
	docker build . --file Dockerfile --tag $(DOCKER_TAG)
