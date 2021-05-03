#!/bin/bash

./sniffer -mode sync -count 200
sleep 2

./sniffer -mode identify_date
