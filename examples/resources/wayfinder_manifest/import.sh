#!/bin/sh
set -eu
# <group>/<Kind>/[workspace/[environment/]]name[@version]
terraform import wayfinder_manifest.developers 'org/Group/developers'
