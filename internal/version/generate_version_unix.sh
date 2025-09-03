#!/usr/bin/env bash

if [[ ! -f VERSION ]]; then
  echo -n "0.0.0" >VERSION
  echo "Generated VERSION file with the default version...."
fi

COMMIT="$(git rev-parse HEAD)"
echo -n "${COMMIT}" >COMMIT
echo "Generated COMMIT file with the current commit hash...."
