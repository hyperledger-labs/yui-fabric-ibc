#!/usr/bin/env bash

function clearContainers() {
  CONTAINER_IDS=$(docker ps -a | awk '($2 ~ /dev-peer.*/) {print $1}')
  if [[ -z "$CONTAINER_IDS" || "$CONTAINER_IDS" == " " ]]; then
    echo "---- No containers available for deletion ----"
  else
    docker rm -f ${CONTAINER_IDS}
  fi
}

function removeUnwantedImages() {
  DOCKER_IMAGE_IDS=$(docker images | awk '($1 ~ /dev-peer.*/) {print $3}')
  if [[ -z "$DOCKER_IMAGE_IDS" || "$DOCKER_IMAGE_IDS" == " " ]]; then
    echo "---- No images available for deletion ----"
  else
    docker rmi -f ${DOCKER_IMAGE_IDS}
  fi
}

docker-compose down --volumes --remove-orphans
clearContainers
removeUnwantedImages

rm -rf ./artifacts/*.block ./artifacts/*.tx
rm -rf ./dev.block
rm -rf ./*.tar.gz
