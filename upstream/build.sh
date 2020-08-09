#!/bin/bash

# Build some files we need to to update the internal embedded copy of tailwindcss.
# This is only used/needed by the maintainer of this project.

# Dependencies:
# - node, npm, npx

mkdir -p out
mkdir -p out/tests

# run them all, ignore errors
for n in `find in -type f`; do
    outn=`echo $n|sed 's/in\//out\//g'`
    echo "Processing $n => $outn"
    npx tailwindcss build $n -o $outn
done
