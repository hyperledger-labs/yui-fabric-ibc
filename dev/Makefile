IMAGE_TAG?=2.1.1
DOCKER_COMPOSE?=docker-compose

.PHONY: init
init:
	rm -rf ./artifacts
	docker run --rm \
	--workdir="/root/fabibc" \
	--volume $(CURDIR)/sh:/root/fabibc/sh \
	--volume $(CURDIR)/config:/root/fabibc/config \
	--volume $(CURDIR)/artifacts:/root/fabibc/artifacts \
	hyperledger/fabric-tools:${IMAGE_TAG} \
	bash -c "./sh/init.sh"

.PHONY: up
up:
	IMAGE_TAG=$(IMAGE_TAG) docker-compose up -d

.PHONY: down
down:
	./sh/clean.sh

CC_NAME=fabibc
.PHONY: vendor
vendor:
	cd ../cmd/$(CC_NAME) && go mod vendor

.PHONY: start
start: down init vendor up
	$(DOCKER_COMPOSE) exec cli bash -c "./sh/deployCC.sh"
	$(DOCKER_COMPOSE) logs -f

CC_VERSION?=1
updateCC: vendor
	$(DOCKER_COMPOSE) exec --env VERSION=$(CC_VERSION) cli bash -c "./sh/deployCC.sh"
