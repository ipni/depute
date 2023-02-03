package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/ipfs/go-log/v2"
	"github.com/ipni/depute"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var logger = log.Logger("depute/cmd")

const (
	libp2pUserAgent = "ipni/depute"
)

func main() {
	libp2pIdentityPath := flag.String("libp2pIdentityPath", "", "The path to the marshalled libp2p host identity. If unspecified a random identity is generated.")
	libp2pListenAddrs := flag.String("libp2pListenAddrs", "", "The comma separated libp2p host listen addrs. If unspecified the default listen addrs are used at ephemeral port.")
	retrievalAddrs := flag.String("retrievalAddrs", "", "The comma separated retrieval multiaddrs to advertise. If unspecified, libp2p host listen addrs are used.")
	grpcListenAddr := flag.String("grpcListenAddr", "0.0.0.0:40080", "The gRPC server listen address.")
	grpcTlsCertPath := flag.String("grpcTlsCertPath", "", "The path to gRPC server TLS Certificate.")
	grpcTlsKeyPath := flag.String("grpcTlsKeyPath", "", "The path to gRPC server TLS Key.")
	logLevel := flag.String("logLevel", "info", "The logging level. Only applied if GOLOG_LOG_LEVEL environment variable is unset.")
	flag.Parse()

	if _, set := os.LookupEnv("GOLOG_LOG_LEVEL"); !set {
		_ = log.SetLogLevel("*", *logLevel)
	}

	hOpts := []libp2p.Option{
		libp2p.UserAgent(libp2pUserAgent),
	}
	if *libp2pIdentityPath != "" {
		p := filepath.Clean(*libp2pIdentityPath)
		logger := logger.With("path", p)
		logger.Info("Unmarshalling libp2p host identity")
		mid, err := os.ReadFile(p)
		if err != nil {
			logger.Fatalw("Failed to read libp2p host identity file", "err", err)
		}
		id, err := crypto.UnmarshalPrivateKey(mid)
		if err != nil {
			logger.Fatalw("Failed to unmarshal libp2p host identity file", "err", err)
		}
		hOpts = append(hOpts, libp2p.Identity(id))
	}
	if *libp2pListenAddrs != "" {
		hOpts = append(hOpts, libp2p.ListenAddrStrings(strings.Split(*libp2pListenAddrs, ",")...))
	}
	h, err := libp2p.New(hOpts...)
	if err != nil {
		logger.Fatalw("Failed to instantiate libp2p host", "err", err)
	}

	deputeOpts := []depute.Option{
		depute.WithHost(h),
		depute.WithGrpcListenAddr(*grpcListenAddr),
	}
	if *retrievalAddrs != "" {
		rAddrs := strings.Split(*libp2pListenAddrs, ",")
		deputeOpts = append(deputeOpts, depute.WithRetrievalAddrs(rAddrs...))
	}

	var gsOpts []grpc.ServerOption
	// TODO: expose more flags for gRPC server options.
	if *grpcTlsCertPath != *grpcTlsKeyPath {
		if *grpcTlsCertPath == "" || *grpcTlsKeyPath == "" {
			logger.Fatal("Both TLS Certificate and Key path must be specified.")
		} else {
			creds, err := credentials.NewServerTLSFromFile(*grpcTlsCertPath, *grpcTlsKeyPath)
			if err != nil {
				logger.Fatalw("Failed to instantiate server TLS credentials", "err", err)
			}
			gsOpts = append(gsOpts, grpc.Creds(creds))
		}
	}
	deputeOpts = append(deputeOpts, depute.WithGrpcServerOptions(gsOpts...))

	c, err := depute.New(deputeOpts...)
	if err != nil {
		logger.Fatalw("Failed to instantiate depute", "err", err)
	}
	ctx := context.Background()
	if err := c.Start(ctx); err != nil {
		logger.Fatalw("Failed to start depute", "err", err)
	}
	sch := make(chan os.Signal, 1)
	signal.Notify(sch, os.Interrupt)

	<-sch
	logger.Info("Terminating...")
	if err := c.Shutdown(ctx); err != nil {
		logger.Warnw("Failure occurred while shutting down server.", "err", err)
	} else {
		logger.Info("Shut down server successfully.")
	}
}
