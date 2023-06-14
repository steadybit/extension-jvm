#!/bin/sh
#
# Copyright 2022 steadybit GmbH. All rights reserved.
#

dlv --listen=:40000 --headless=true --api-version=2 --accept-multiclient exec /opt/steadybit/extension/extension
