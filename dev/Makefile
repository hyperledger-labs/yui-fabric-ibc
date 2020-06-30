CC_NAME=fabibc

.PHONY: init
init:
	rm -rf ./artifacts
	./sh/init.sh

.PHONY: up
up:
	docker-compose up -d

.PHONY: down
down:
	./sh/clean.sh

.PHONY: vendor
vendor:
	cd ../cmd/$(CC_NAME) && go mod vendor

.PHONY: start
start: down init vendor up
	docker exec -it cli bash -c "./sh/deployCC.sh"
	docker-compose logs -f

CC_VERSION?=1
updateCC: vendor
	docker exec -it --env VERSION=$(CC_VERSION) cli bash -c "./sh/deployCC.sh"


