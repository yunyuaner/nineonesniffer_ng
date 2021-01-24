#!/bin/bash

#video_id="423117"
#video_parts_count=49

video_id=$1
video_parts_count=$2

mkdir -p ./video_parts/${video_id}

for ((i = 0; i <= ${video_parts_count}; i++)); do
	wget -O "./video_parts/${video_id}/${video_id}${i}.ts" "https://cdn.91p07.com//m3u8/${video_id}/${video_id}${i}.ts"
done
