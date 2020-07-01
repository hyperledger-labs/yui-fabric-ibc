#!/bin/bash

IMAGE=hyperledger/fabric-tools:2.1
FIXTURES_DIR=${PWD}/tests/fixtures
CONF_NAME=crypto-config-test.yaml

echo ${PWD}

if [ -d "${FIXTURES_DIR}/organizations/peerOrganizations" ]; then
    rm -Rf ${FIXTURES_DIR}/organizations/peerOrganizations && rm -Rf ${FIXTURES_DIR}/organizations/ordererOrganizations
fi

set -x
docker container run -v ${FIXTURES_DIR}:/wd -it ${IMAGE} \
    cryptogen generate --config=/wd/${CONF_NAME} --output "/wd/organizations"
res=$?
set +x
if [ $res -ne 0 ]; then
    echo "Failed to generate certificates..."
    exit 1
fi

# pick each org's peer msp
# note: MSP ID should be specified in configtx.yaml in practice
len=$(yq r ${FIXTURES_DIR}/${CONF_NAME} --length 'PeerOrgs')
for i in $(seq 0 $(expr ${len} - 1)); do
    dir=$(yq r ${FIXTURES_DIR}/${CONF_NAME} 'PeerOrgs.['${i}'].Domain')
    mspid=$(yq r ${FIXTURES_DIR}/${CONF_NAME} 'PeerOrgs.['${i}'].Name')MSP
    # HACK peer0
    mv ${FIXTURES_DIR}/organizations/peerOrganizations/${dir}/peers/peer0.${dir}/msp ${FIXTURES_DIR}/organizations/peerOrganizations/${mspid}
    rm -rf ${FIXTURES_DIR}/organizations/peerOrganizations/${dir}
done

exit 0
