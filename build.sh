#!/bin/sh

# cgo shared lib for c client
go build -buildmode=c-shared -o xdag_libp2p.so xdag_libp2p.go

# golang server
go build -o xdag_libp2p_server xdag_libp2p_server.go

# c client
gcc -o xdag_libp2p_client xdag_libp2p_client.c xdag_libp2p.so