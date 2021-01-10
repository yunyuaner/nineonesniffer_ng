#!/bin/bash

cmd="cat"
for ((i = 0; i <= 60; i++)); do
    cmd="${cmd} 424080${i}.ts "
done
cmd="${cmd} > 424080.ts"

echo $cmd
