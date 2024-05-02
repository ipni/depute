package depute

import (
	"fmt"
	"net/url"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
	"github.com/ipfs/go-datastore/sync"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/storage/dsadapter"
	"github.com/ipni/go-libipni/dagsync"
	"github.com/ipni/go-libipni/dagsync/ipnisync"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/multiformats/go-multiaddr"
	"google.golang.org/grpc"
)

type (
	Option  func(*options) error
	options struct {
		announceToURLs   []*url.URL
		httpListenAddr   string
		noPubsubAnnounce bool
		publishAddrs     []multiaddr.Multiaddr

		ds             datastore.Batching
		grpcListenAddr string
		grpcServerOpts []grpc.ServerOption
		h              host.Host
		ls             *ipld.LinkSystem
		retrievalAddrs []string
		publisher      dagsync.Publisher
		pubTopicName   string
	}
)

func newOptions(o ...Option) (*options, error) {
	opts := options{
		grpcListenAddr: "0.0.0.0:40080",
		pubTopicName:   "/indexer/ingest/mainnet",
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
		privKey := opts.h.Peerstore().PrivKey(opts.h.ID())
		opts.publisher, err = ipnisync.NewPublisher(*opts.ls, privKey,
			ipnisync.WithStreamHost(opts.h),
			ipnisync.WithHeadTopic(opts.pubTopicName),
			ipnisync.WithHTTPListenAddrs(opts.httpListenAddr),
		)
		if err != nil {
			return nil, err
		}
	}
	if opts.noPubsubAnnounce && len(opts.announceToURLs) == 0 {
		// No pubsub or HTTP announcements, so no publication to announce.
		opts.publishAddrs = nil
	} else if len(opts.publishAddrs) == 0 {
		opts.publishAddrs = append(opts.publishAddrs, opts.publisher.Addrs()...)
		logger.Warn("No advertisement publication address to put into announcements. Using publisher host addresses, but external address may be needed.", "addrs", opts.publishAddrs)
	}

	return &opts, nil
}

// WithAnnounceToURLs sets URLs of indexers to send direct HTTP
// announcements to.
func WithAnnounceToURLs(urls []string) Option {
	return func(o *options) error {
		for _, ustr := range urls {
			u, err := url.Parse(ustr)
			if err != nil {
				return err
			}
			o.announceToURLs = append(o.announceToURLs, u)
		}
		return nil
	}
}

// WithPublishAddrs sets the addresses put into announcements to tell indexers
// where to get the advertisements.
func WithPublishAddrs(addrs []multiaddr.Multiaddr) Option {
	return func(o *options) error {
		o.publishAddrs = addrs
		return nil
	}
}

// WithHttpListenAddrs sets the address to listen on for publishing
// advertisements over HTTP.
func WithHttpListenAddr(addr string) Option {
	return func(o *options) error {
		o.httpListenAddr = addr
		return nil
	}
}

func WithHost(h host.Host) Option {
	return func(o *options) error {
		o.h = h
		return nil
	}
}

// WithNoPubsubAnnounce disables libp2p pubsub announcements.
func WithNoPubsubAnnounce() Option {
	return func(o *options) error {
		o.noPubsubAnnounce = true
		return nil
	}
}

func WithPublisher(publisher dagsync.Publisher) Option {
	return func(o *options) error {
		o.publisher = publisher
		return nil
	}
}

func WithPublishTopic(topicName string) Option {
	return func(o *options) error {
		o.pubTopicName = topicName
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
