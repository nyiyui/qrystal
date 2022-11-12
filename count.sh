#!/bin/sh

find . -type f \
	| grep -v '\./pkg' \
	| grep -v '\.git' \
	| grep -v '\.zst$' \
	| grep -v 'build2' \
	| xargs wc -l \
	| sort -nr
