package depute

import (
	"context"
	"net"

	"github.com/gogo/status"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-log/v2"
	"github.com/ipld/go-ipld-prime"
	_ "github.com/ipld/go-ipld-prime/codec/dagcbor"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	depute "github.com/ipni/depute/api/v0"
	"github.com/ipni/index-provider/engine/chunker"
	"github.com/ipni/storetheindex/api/v0/ingest/schema"
	"github.com/multiformats/go-multicodec"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var (
	logger = log.Logger("depute")

	_ depute.PublisherServer = (*Depute)(nil)

	linkPrototype = cidlink.LinkPrototype{
		Prefix: cid.Prefix{
			Version:  1,
			Codec:    uint64(multicodec.DagCbor),
			MhType:   uint64(multicodec.Sha2_256),
			MhLength: -1,
		},
	}

	dsKeyLatestAdLink = datastore.NewKey("depute/latestAdLink")
)

type Depute struct {
	*options
	chunker *chunker.ChainChunker
	server  *grpc.Server
}

func New(o ...Option) (*Depute, error) {
	opts, err := newOptions(o...)
	if err != nil {
		return nil, err
	}
	return &Depute{
		options: opts,
		server:  grpc.NewServer(opts.grpcServerOpts...),
	}, nil
}

func (d *Depute) NotifyContent(source depute.Publisher_NotifyContentServer) error {
	chunk, err := d.chunker.Chunk(source.Context(), &notifyContentIter{source: source})
	if err != nil {
		logger.Errorw("Failed to create entries chain chunks", "err", err)
		return status.Errorf(codes.Internal, "failed to create entries chain chunks: %v", err)
	}
	var l depute.Link
	if err := l.Marshal(chunk); err != nil {
		return status.Errorf(codes.Internal, "failed to marshal link: %v", err)
	}
	return source.SendAndClose(&depute.NotifyContent_Response{
		Link: &l,
	})
}

func (d *Depute) Publish(ctx context.Context, req *depute.Publish_Request) (*depute.Publish_Response, error) {
	ad := req.Advertisement
	if ad == nil {
		return nil, status.Error(codes.InvalidArgument, "no advertisement")
	}

	var entries ipld.Link
	if ad.GetEntries() == nil {
		entries = schema.NoEntries
	} else {
		var err error
		entries, err = ad.GetEntries().Unmarshal()
		if err != nil {
			logger.Errorw("Failed to convert entries to link", "err", err)
			return nil, status.Errorf(codes.InvalidArgument, "invalid entries link: %v", err)
		}
	}
	previous, err := d.getLatestAdvertisementLink(ctx)
	if err != nil {
		logger.Errorw("Failed to get latest ad link", "err", err)
		return nil, status.Errorf(codes.Internal, "failed to get latest ad link: %v", err)
	}
	n, err := schema.Advertisement{
		PreviousID: previous,
		Provider:   d.h.ID().String(),
		Addresses:  d.retrievalAddrs,
		Entries:    entries,
		ContextID:  ad.GetContextId(),
		Metadata:   ad.GetMetadata(),
		IsRm:       ad.GetRemoved(),
	}.ToNode()
	if err != nil {
		logger.Errorw("Failed to create IPLD ad node", "err", err)
		return nil, status.Errorf(codes.Internal, "failed to create ad IPLD node: %v", err)
	}
	link, err := d.ls.Store(ipld.LinkContext{Ctx: ctx}, linkPrototype, n)
	if err != nil {
		logger.Errorw("Failed to store ad node", "err", err)
		return nil, status.Errorf(codes.Internal, "failed to store ad IPLD node: %v", err)
	}
	if err := d.setLatestAdvertisementLink(ctx, link); err != nil {
		logger.Errorw("Failed to set latest ad link", "link", link.String(), "err", err)
		return nil, status.Errorf(codes.Internal, "failed to set latest ad link: %v", err)
	}
	logger.Infow("Published advertisement", "link", link.String())
	var l depute.Link
	if err := l.Marshal(link); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal link: %v", err)
	}
	return &depute.Publish_Response{
		Link: &l,
	}, nil
}

func (d *Depute) getLatestAdvertisementLink(ctx context.Context) (ipld.Link, error) {
	v, err := d.ds.Get(ctx, dsKeyLatestAdLink)
	switch err {
	case nil:
		_, c, err := cid.CidFromBytes(v)
		if err != nil {
			return nil, err
		}
		return cidlink.Link{Cid: c}, nil
	case datastore.ErrNotFound:
		return nil, nil
	default:
		return nil, err
	}
}

func (d *Depute) setLatestAdvertisementLink(ctx context.Context, l ipld.Link) error {
	return d.ds.Put(ctx, dsKeyLatestAdLink, l.(cidlink.Link).Bytes())
}

func (d *Depute) Start(_ context.Context) error {
	ln, err := net.Listen("tcp", d.grpcListenAddr)
	if err != nil {
		return err
	}
	depute.RegisterPublisherServer(d.server, d)
	go func() { _ = d.server.Serve(ln) }()
	logger.Infow("Server started", "addr", ln.Addr())
	return nil
}

func (d *Depute) Shutdown(_ context.Context) error {
	d.server.Stop()
	pErr := d.publisher.Close()
	dsErr := d.ds.Close()
	hErr := d.h.Close()
	switch {
	case hErr != nil:
		return hErr
	case dsErr != nil:
		return dsErr
	default:
		return pErr
	}
}
