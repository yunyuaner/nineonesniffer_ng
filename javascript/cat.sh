#!/bin/bash

#video_id="423117"
#video_parts_count=49
video_id=$1
video_parts_count=$2
video_parts_dir="`pwd`/video_parts"
video_merged_dir="`pwd`/video_merged"

cmd="cat"
for ((i = 0; i <= ${video_parts_count}; i++)); do
    cmd="${cmd} ${video_parts_dir}/${video_id}${i}.ts "
done
cmd="${cmd} > ${video_merged_dir}/${video_id}.ts"

eval $cmd

ffmpeg -i ${video_merged_dir}/${video_id}.ts -c:v libx264 -c:a aac -strict -2 ${video_merged_dir}/${video_id}.mp4
