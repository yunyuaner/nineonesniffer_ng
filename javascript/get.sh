#!/bin/bash
for ((i = 0; i <= 60; i++)); do
	wget "https://cdn.91p07.com//m3u8/424080/424080${i}.ts"
done
