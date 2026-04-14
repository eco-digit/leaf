.SILENT:
######################################################################

ifeq ($(OS_CLOUD),)
  OS_CLOUD := openstack
  OS_EXTERNAL_NETWORK_ID := $(OS_EXTERNAL_NETWORK_ID)
endif
ifeq ($(OS_CLOUD),plusserver)
  OS_AUTH_URL := https://scs2.api.pco.get-cloud.io:5000
  OS_EXTERNAL_NETWORK_ID := d051c0bd-510c-4da3-bcf3-d8b7082dd008
endif
ifeq ($(OS_CLOUD),scaleup)
  OS_AUTH_URL := https://keystone.scs1.scaleup.cloud:443
  OS_EXTERNAL_NETWORK_ID := 15227829-b53d-48af-b136-85733999252e
endif
######################################################################

export OS_AUTH_URL
export OS_CLOUD
export OS_EXTERNAL_NETWORK_ID
######################################################################

ANSIBLE_CONFIG := $(CURDIR)/ansible/.ansible.cfg
ANSIBLE_INVENTORY := $(CURDIR)/ansible/inventory_$(OS_CLOUD).ini
ANSIBLE_ROLES_PATH := $(CURDIR)/ansible/roles
DOCKER_DEBIAN_RELEASE := stable			# runtime container
DOCKER_DIR := $(CURDIR)/docker
DOCKER_GOLANG_RELEASE := 1.26-alpine	# builder container
DOCKER_PROMETHEUS_RELEASE := 3.9.1
LEAF_BIN := $(CURDIR)/bin/leaf
LEAF_CONFIG := $(CURDIR)/config/config.yaml.sample
LEAF_DOCKER_TAG := ghcr.io/eco-digit/leaf
LEAF_PROMETHEUS := http://127.0.0.1:9090
LEAF_PROMETHEUS_PASS := passwd
LEAF_PROMETHEUS_USER := admin
TF_IN_AUTOMATION := yes
TF_LOG := INFO
TF_LOG_PATH := $(TOFU_CHDIR)/tofu-$(OS_CLOUD).log
TOFU_CHDIR := $(CURDIR)/opentofu
TOFU_CMD := tofu -chdir=$(TOFU_CHDIR)
TOFU_DEFAULT_ARGS := -input=false -auto-approve -concise -backup="-" -state=$(TOFU_CHDIR)/tofu-$(OS_CLOUD).tfstate
TOFU_DEFAULT_VARS := -var OS_CLOUD=$(OS_CLOUD)
######################################################################

LEAF_DOCKER_BUILD_ARGS := \
	--build-arg DOCKER_DEBIAN_RELEASE=$(DOCKER_DEBIAN_RELEASE) \
	--build-arg DOCKER_GOLANG_RELEASE=$(DOCKER_GOLANG_RELEASE) \
	--build-arg DOCKER_PROMETHEUS_RELEASE=$(DOCKER_PROMETHEUS_RELEASE) \
	--build-arg LEAF_DOCKER_TAG=$(LEAF_DOCKER_TAG) \
	--build-arg LEAF_PROMETHEUS=$(LEAF_PROMETHEUS) \
	--build-arg LEAF_PROMETHEUS_PASS=$(LEAF_PROMETHEUS_PASS) \
	--build-arg LEAF_PROMETHEUS_USER=$(LEAF_PROMETHEUS_USER)
######################################################################

export DOCKER_DEBIAN_RELEASE
export DOCKER_DIR
export DOCKER_GOLANG_RELEASE
export DOCKER_PROMETHEUS_RELEASE
export LEAF_DOCKER_TAG
export LEAF_BIN
export LEAF_CONFIG
export LEAF_DOCKER_BUILD_ARGS
export LEAF_PROMETHEUS
export LEAF_PROMETHEUS_PASS
export LEAF_PROMETHEUS_USER
######################################################################

check-env-var-%:
	@ if [ "${${*}}" = "" ]; then \
	  echo "Environment variable $* not set"; \
	  exit 1; \
	fi
######################################################################

all build leaf:
	go build -ldflags '-s -w' -o $(LEAF_BIN) cmd/leaf/main.go

clean:
	docker image rm -f $(LEAF_DOCKER_TAG) 2>/dev/null
	rm -f $(LEAF_BIN)

image:
	docker build -f $(DOCKER_DIR)/Dockerfile -t $(LEAF_DOCKER_TAG) $(LEAF_DOCKER_BUILD_ARGS) .

infra infra-deploy: tofu
	$(TOFU_CMD) apply $(TOFU_DEFAULT_ARGS) $(TOFU_DEFAULT_VARS) -var PUBLIC_NETWORK_ID=$(OS_EXTERNAL_NETWORK_ID)

infra-setup:
	ANSIBLE_CONFIG=$(ANSIBLE_CONFIG) ANSIBLE_ROLES_PATH=$(ANSIBLE_ROLES_PATH) \
		ansible-playbook -i $(ANSIBLE_INVENTORY) $(CURDIR)/ansible/playbook.yml

infra-destroy:
	$(TOFU_CMD) destroy $(TOFU_DEFAULT_ARGS) $(TOFU_DEFAULT_VARS) -var PUBLIC_NETWORK_ID=$(OS_EXTERNAL_NETWORK_ID)
	find $(CURDIR)/opentofu -name "tofu-$(OS_CLOUD).tfstate" -delete
	rm -f "$(ANSIBLE_INVENTORY)" 2>/dev/null

run run-leaf: all
	$(LEAF_BIN) --config $(LEAF_CONFIG)

run-image: image
	docker run --rm --name leaf -it \
		-p 9010:9010 \
		-p 9090:9090 \
		$(LEAF_DOCKER_TAG)

test test-image: image
	docker build -f $(DOCKER_DIR)/Dockerfile_test -t $(LEAF_DOCKER_TAG)_test $(LEAF_DOCKER_BUILD_ARGS) .
	docker run --rm --name leaf_test -it -d \
		-p 9010:9010 \
		-p 9090:9090 \
		$(LEAF_DOCKER_TAG)
	sleep 7
	docker exec leaf_test env | sort
	docker container stop leaf_test

tofu tofu-check tofu-init tofu-validate:
	$(TOFU_CMD) init -upgrade
	$(TOFU_CMD) fmt -check
	$(TOFU_CMD) validate -compact-warnings

ssh:
	@if [ ! -f "$(ANSIBLE_INVENTORY)" ]; then \
		echo "Error: Inventory file $(ANSIBLE_INVENTORY) not found."; \
		exit 1; \
	fi; \
	SSH_ARGS=$$(grep ssh_args $(ANSIBLE_CONFIG) | cut -d'=' -f2- | xargs); \
	SSH_USER=$$(grep ansible_user= ansible/inventory_$(OS_CLOUD).ini | sed 's/.*ansible_user=\([^ ]*\).*/\1/'); \
	SSH_HOST=$$(grep ansible_host= ansible/inventory_$(OS_CLOUD).ini | sed 's/.*ansible_host=\([^ ]*\).*/\1/'); \
	if [ -z "$$SSH_USER" ] || [ -z "$$SSH_HOST" ]; then \
		echo "Error: Could not extract ansible_user or ansible_host from inventory"; \
		exit 1; \
	fi; \
	echo "Connecting to $$SSH_HOST ..."; \
	ssh $$SSH_ARGS -l $$SSH_USER $$SSH_HOST
