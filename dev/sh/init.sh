#!/usr/bin/env bash
set -eux

ORG_NAME=SampleOrg
CHANNEL_NAME=dev
export FABRIC_CFG_PATH=${PWD}/config

configtxgen \
-profile OrdererGenesis \
-channelID system-channel \
-outputBlock ./artifacts/orderer.block

configtxgen \
-profile SampleSingleMSPSolo \
-channelID ${CHANNEL_NAME} \
-outputCreateChannelTx ./artifacts/${CHANNEL_NAME}.tx

configtxgen \
-profile SampleSingleMSPSolo \
-channelID ${CHANNEL_NAME} \
-outputAnchorPeersUpdate ./artifacts/${ORG_NAME}Anchors.tx \
-asOrg ${ORG_NAME}
