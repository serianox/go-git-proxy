SHELL := /bin/bash

%: %.go
	go build -o $@ $<

main: main.go

build: main

test: build
	-@ <main.pid xargs kill 2>/dev/null || true
	-@ <nginx.pid xargs kill 2>/dev/null || true
	nginx -c nginx.conf -p .
	./main -tcp=127.0.0.1:8081 scm https://github.com & echo $$! >main.pid
	@ sleep 1
	git ls-remote http://127.0.0.1:8080/scm/serianox/pygments.git || true
	-@ <main.pid xargs kill 2>/dev/null || true
	-@ <nginx.pid xargs kill 2>/dev/null || true
