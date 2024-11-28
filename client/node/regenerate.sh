#!/bin/env bash
# Path to this plugin
PROTOC_GEN_TS_PATH="./node_modules/.bin/protoc-gen-ts"
PROTOC_GEN_JS_PATH="./node_modules/.bin/protoc-gen-js"

# Directory to write generated code to (.js and .d.ts files)
OUT_DIR="./base"
    # --js_out=import_style="commonjs,binary:${OUT_DIR}" \
protoc \
    -I=. \
    --plugin="protoc-gen-ts=${PROTOC_GEN_TS_PATH}" \
    --plugin="protoc-gen-js=${PROTOC_GEN_JS_PATH}" \
    --ts_out="${OUT_DIR}" \
    --js_out="${OUT_DIR}" \
    base.proto