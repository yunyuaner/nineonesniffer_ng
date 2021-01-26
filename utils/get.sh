#!/bin/bash

source "`pwd`/../utils/path.sh"

protocol="https"
base_url="${protocol}://cdn.91p07.com//m3u8"

video_id=$1
video_parts_count=$2
timeout=120

mkdir -p "${video_parts_dir}/${video_id}"

for ((i = 0; i <= ${video_parts_count}; i++)); do
    video_part_name="${video_parts_dir}/${video_id}/${video_id}${i}.ts"
    if ! [ -f $video_part_name ]; then
	    wget -O "${video_parts_dir}/${video_id}/${video_id}${i}.ts" --timeout ${timeout} "${base_url}/${video_id}/${video_id}${i}.ts"
    fi
done
