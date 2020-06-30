#!/usr/bin/env bash
set -eux

FABRIC_CFG_PATH=${PWD}/config
CHANNEL_NAME=dev
CC_NAME=${CC_NAME:="fabibc"}
CC_SRC_PATH=/opt/gopath/src/chaincodedev/chaincode/${CC_NAME}
CC_RUNTIME_LANGUAGE=golang
VERSION=${VERSION:="1"}

echo "### Deploy Chaincode ${CC_NAME}"
peer lifecycle chaincode package ${CC_NAME}.tar.gz --path ${CC_SRC_PATH} --lang ${CC_RUNTIME_LANGUAGE} --label ${CC_NAME}_${VERSION}
peer lifecycle chaincode install ${CC_NAME}.tar.gz

peer lifecycle chaincode queryinstalled >&log.txt
cat log.txt
PACKAGE_ID=$(sed -n "/${CC_NAME}_${VERSION}/{s/^Package ID: //; s/, Label:.*$//; p;}" log.txt)

peer lifecycle chaincode approveformyorg -o orderer.dev.com:7050 --channelID ${CHANNEL_NAME} --name ${CC_NAME} --version ${VERSION} --sequence ${VERSION} --init-required --package-id ${PACKAGE_ID}
peer lifecycle chaincode checkcommitreadiness --channelID ${CHANNEL_NAME} --name ${CC_NAME} --version ${VERSION} --sequence ${VERSION} --output json --init-required

peer lifecycle chaincode commit -o orderer.dev.com:7050 --channelID ${CHANNEL_NAME} --name ${CC_NAME} --version ${VERSION} --sequence ${VERSION} --init-required --peerAddresses peer0.peerorg.dev.com:7051
peer lifecycle chaincode querycommitted --channelID ${CHANNEL_NAME} --name ${CC_NAME}

peer chaincode invoke -o orderer.dev.com:7050 -C ${CHANNEL_NAME} -n ${CC_NAME} --isInit -c '{"Args":["initChaincode", "{}"]}'

echo "### Finished to deploy Chaincode ${CC_NAME}"
