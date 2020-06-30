#!/bin/bash
# Copyright London Stock Exchange Group All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
set -eux
CHANNEL_NAME=dev

# This script expedites the chaincode development process by automating the
# requisite channel create/join commands

# We use a pre-generated orderer.block and channel transaction artifact (dev.tx),
# both of which are created using the configtxgen tool

# first we create the channel against the specified configuration in devtx
# this call returns a channel configuration block - dev.block - to the CLI container
echo "### Creating channel ${CHANNEL_NAME}"
peer channel create -c ${CHANNEL_NAME} -f ./artifacts/${CHANNEL_NAME}.tx -o orderer.dev.com:7050

# now we will join the channel and start the chain with dev.block serving as the
# channel's first block (i.e. the genesis block)
peer channel join -b ./${CHANNEL_NAME}.block  -o orderer.dev.com:7050

# Now the user can proceed to build and start chaincode in one terminal
# And leverage the CLI container to issue install instantiate invoke query commands in another

#we should have bailed if above commands failed.
#we are here, so they worked

sleep 600000
exit 0
