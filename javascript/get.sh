#!/bin/bash

video_id="426749"
video_parts_count=74

for ((i = 0; i <= ${video_parts_count}; i++)); do
	wget -O "./video_parts/${video_id}${i}.ts" "https://cdn.91p07.com//m3u8/${video_id}/${video_id}${i}.ts"
done
