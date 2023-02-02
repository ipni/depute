package depute

import (
	"fmt"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
	"github.com/ipfs/go-datastore/sync"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/storage/dsadapter"
	"github.com/ipni/storetheindex/dagsync"
	"github.com/ipni/storetheindex/dagsync/dtsync"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/multiformats/go-multiaddr"
	"google.golang.org/grpc"
)

type (
	Option  func(*options) error
	options struct {
		h              host.Host
		ds             datastore.Batching
		ls             *ipld.LinkSystem
		retrievalAddrs []string
		publisher      dagsync.Publisher
		grpcListenAddr string
		grpcServerOpts []grpc.ServerOption
	}
)

func newOptions(o ...Option) (*options, error) {
	opts := options{
		grpcListenAddr: "0.0.0.0:40080",
	}
	for _, apply := range o {
		if err := apply(&opts); err != nil {
			return nil, err
		}
	}

	var err error
	if opts.h == nil {
		opts.h, err = libp2p.New()
		if err != nil {
			return nil, err
		}
	}
	if opts.ds == nil {
		opts.ds = sync.MutexWrap(datastore.NewMapDatastore())
	}
	if opts.ls == nil {
		ls := cidlink.DefaultLinkSystem()
		store := &dsadapter.Adapter{
			Wrapped: namespace.Wrap(opts.ds, datastore.NewKey("ls")),
		}
		ls.SetReadStorage(store)
		ls.SetWriteStorage(store)
		opts.ls = &ls
	}
	if len(opts.retrievalAddrs) == 0 {
		addrs := opts.h.Addrs()
		opts.retrievalAddrs = make([]string, 0, len(addrs))
		for _, addr := range addrs {
			opts.retrievalAddrs = append(opts.retrievalAddrs, addr.String())
		}
	}
	if opts.publisher == nil {
		pds := namespace.Wrap(opts.ds, datastore.NewKey("pub"))
		opts.publisher, err = dtsync.NewPublisher(opts.h, pds, *opts.ls, "/indexer/ingest/mainnet")
		if err != nil {
			return nil, err
		}
	}
	return &opts, nil
}

func WithHost(h host.Host) Option {
	return func(o *options) error {
		o.h = h
		return nil
	}
}

func WithRetrievalAddrs(a ...string) Option {
	return func(o *options) error {
		for _, addr := range a {
			_, err := multiaddr.NewMultiaddr(addr)
			if err != nil {
				return fmt.Errorf("invalid retrieval multiaddr: %w", err)
			}
		}
		o.retrievalAddrs = a
		return nil
	}
}

func WithGrpcListenAddr(a string) Option {
	return func(o *options) error {
		o.grpcListenAddr = a
		return nil
	}
}

func WithGrpcServerOptions(opt ...grpc.ServerOption) Option {
	return func(o *options) error {
		o.grpcServerOpts = opt
		return nil
	}
}
