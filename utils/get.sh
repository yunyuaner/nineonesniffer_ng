#!/bin/bash

source "`pwd`/../utils/path.sh"

protocol="https"
base_url="${protocol}://cdn.91p07.com//m3u8"

video_id=$1
video_parts_count=$2

mkdir -p "${video_parts_dir}/${video_id}"

for ((i = 0; i <= ${video_parts_count}; i++)); do
	wget -O "${video_parts_dir}/${video_id}/${video_id}${i}.ts" "${base_url}/${video_id}/${video_id}${i}.ts"
done
