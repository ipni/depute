package depute

import (
	"errors"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
)

func (l *Link) Unmarshal() (ipld.Link, error) {
	if len(l.Value) == 0 {
		return cidlink.Link{Cid: cid.Undef}, nil
	}
	read, c, err := cid.CidFromBytes(l.Value)
	if err != nil {
		return nil, err
	}
	if read != len(l.Value) {
		return nil, errors.New("bytes remain after decoding link")
	}
	return cidlink.Link{Cid: c}, nil
}

func (l *Link) Marshal(lnk ipld.Link) error {
	if lnk == nil {
		l.Value = nil
		return nil
	}
	l.Value = lnk.(cidlink.Link).Cid.Bytes()
	return nil
}
