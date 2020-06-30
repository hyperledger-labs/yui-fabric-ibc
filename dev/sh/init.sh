#!/usr/bin/env bash
set -eux

CHANNEL_NAME=dev
export FABRIC_CFG_PATH=${PWD}/config

configtxgen -profile OrdererGenesis -channelID system-channel -outputBlock ./artifacts/orderer.block
configtxgen -profile SampleSingleMSPSolo -outputCreateChannelTx ./artifacts/${CHANNEL_NAME}.tx -channelID ${CHANNEL_NAME}
configtxgen -profile SampleSingleMSPSolo -outputAnchorPeersUpdate ./artifacts/SampleOrgAnchors.tx -channelID ${CHANNEL_NAME} -asOrg SampleOrg
