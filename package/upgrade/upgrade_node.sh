#!/bin/bash -ex

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

until $SCRIPT_DIR/do_upgrade_node.sh $@; do
  if [ "$1" = "prepare" ]; then
    exit 1
  fi

  if [ -e "/tmp/skip-retry-with-fail" ]; then
    exit 1
  fi

  if [ -e "/tmp/skip-retry-with-succeed" ]; then
    exit 0
  fi

  echo "Running \"upgrade_node.sh $@\" errors, will retry..."
  sleep 30
done
