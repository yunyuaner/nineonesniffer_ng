#!/bin/bash

source "`pwd`/../utils/path.sh"

video_id=$1
video_parts_count=$2
#video_parts_dir="`pwd`/video_parts"
#video_merged_dir="`pwd`/video_merged"

cmd="cat"
for ((i = 0; i <= ${video_parts_count}; i++)); do
    cmd="${cmd} ${video_parts_dir}/${video_id}/${video_id}${i}.ts "
done
cmd="${cmd} > ${video_merged_dir}/${video_id}.ts"

eval $cmd

if [ -f "${video_merged_dir}/${video_id}.mp4" ]; then
    rm -rf "${video_merged_dir}/${video_id}.mp4"
fi

if [ "X$3" == "Xtranscode" ]; then
    ffmpeg -i ${video_merged_dir}/${video_id}.ts -c:v libx264 -c:a aac -strict -2 ${video_merged_dir}/${video_id}.mp4
fi
