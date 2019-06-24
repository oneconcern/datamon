#! /bin/sh

BIN="$(pwd)/$1"
CPU_PROF="$(pwd)/$2"
MEM_PROF="$(pwd)/$3"
OUT_DIR="$4"

(cd "$OUT_DIR" && \
     go tool pprof -svg "$BIN" "$CPU_PROF" && \
     mv profile001.svg cpu_prof.svg)

upload_mem_prof="${MEM_PROF}/upload_bundle.mem.prof"
(cd "$OUT_DIR" && \
     go tool pprof -svg "$BIN" "$upload_mem_prof" && \
     mv profile001.svg mem_prof.svg)
