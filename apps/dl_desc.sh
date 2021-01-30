#!/bin/bash

if [ $# -eq 0 ]; then
    echo "Please provide video page url first"
    exit 0
fi

./sniffer -mode dl_desc -url "$1"
