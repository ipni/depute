# :tophat: depute

[![Go Test](https://github.com/ipni/depute/actions/workflows/go-test.yml/badge.svg)](https://github.com/ipni/depute/actions/workflows/go-test.yml)

A gRPC service to advertise content addressable data onto IPNI.

## Install

To install `depute` CLI directly via Golang, run:

```shell
$ go install github.com/ipni/depute/cmd/depute@latest
```

## Usage

```shell
$ depute -h 
Usage of depute:
Usage of ./depute:
  -directAnnounceURL value
    	Indexer URL to send direct http announcement to. Multiple OK
  -grpcListenAddr string
    	The gRPC server listen address. (default "0.0.0.0:40080")
  -grpcTlsCertPath string
    	Path to gRPC server TLS Certificate.
  -grpcTlsKeyPath string
    	Path to gRPC server TLS Key.
  -httpListenAddr string
    	Address to listen on for publishing advertisements over HTTP.
  -libp2pIdentityPath string
    	Path to the marshalled libp2p host identity. If unspecified a random identity is generated.
  -libp2pListenAddrs string
    	Comma separated libp2p host listen addrs. If unspecified the default listen addrs are used at ephemeral port.
  -logLevel string
    	Logging level. Only applied if GOLOG_LOG_LEVEL environment variable is unset. (default "info")
  -noPubsub
    	Disable pubsub announcements of new advertisements.
  -pubAddr value
    	Address to tell indexer where to retrieve advertisements. Multiple OK
  -retrievalAddrs string
    	Comma separated retrieval multiaddrs to advertise. If unspecified, libp2p host listen addrs are used.
  -topic string
    	Sets the topic that pubsub messages are send on. (default "/indexer/ingest/mainnet")
```

### Run Server Locally

To run the `depute` HTTP server locally, execute:

```shell
$ go run ./cmd/depute
```

The above command starts the gRPC server exposed on default listen address: `http://localhost:40080`.

To shutdown the server, interrupt the terminal by pressing `Ctrl + C`

## License

[SPDX-License-Identifier: Apache-2.0 OR MIT](LICENSE.md)
