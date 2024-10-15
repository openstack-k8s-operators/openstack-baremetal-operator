#!/bin/bash
set -ex

oc delete validatingwebhookconfiguration/vopenstackbaremetalset.kb.io --ignore-not-found
oc delete validatingwebhookconfiguration/vopenstackprovisionserver.kb.io --ignore-not-found
oc delete mutatingwebhookconfiguration/mopenstackprovisionserver.kb.io --ignore-not-found
