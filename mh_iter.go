package depute

import (
	depute "github.com/ipni/depute/api/v0"
	provider "github.com/ipni/index-provider"
	"github.com/multiformats/go-multihash"
)

var _ provider.MultihashIterator = (*notifyContentIter)(nil)

type notifyContentIter struct {
	source depute.Publisher_NotifyContentServer
}

func (i *notifyContentIter) Next() (multihash.Multihash, error) {
	req, err := i.source.Recv()
	if err != nil {
		return nil, err
	}
	return req.Multihash.GetValue(), err
}
