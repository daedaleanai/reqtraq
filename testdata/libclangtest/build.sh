#!/bin/bash -e

cd "$(dirname "$0")"

rm -rf build
cmake -S . -B build -G Ninja -DCMAKE_EXPORT_COMPILE_COMMANDS=ON
