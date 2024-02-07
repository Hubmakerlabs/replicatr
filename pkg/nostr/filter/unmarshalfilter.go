package filter

//go:generate go run unmarshalgen.go

import (
	"fmt"

	"github.com/Hubmakerlabs/replicatr/pkg/nostr/kinds"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/tag"
	"github.com/Hubmakerlabs/replicatr/pkg/nostr/timestamp"
)

type UnmarshalingFilter struct {
	IDs     tag.T         `json:"ids,omitempty"`
	Kinds   kinds.T       `json:"kinds,omitempty"`
	Authors tag.T         `json:"authors,omitempty"`
	Ta      tag.T         `json:"#a,omitempty"`
	Tb      tag.T         `json:"#b,omitempty"`
	Tc      tag.T         `json:"#c,omitempty"`
	Td      tag.T         `json:"#d,omitempty"`
	Te      tag.T         `json:"#e,omitempty"`
	Tf      tag.T         `json:"#f,omitempty"`
	Tg      tag.T         `json:"#g,omitempty"`
	Th      tag.T         `json:"#h,omitempty"`
	Ti      tag.T         `json:"#i,omitempty"`
	Tj      tag.T         `json:"#j,omitempty"`
	Tk      tag.T         `json:"#k,omitempty"`
	Tl      tag.T         `json:"#l,omitempty"`
	Tm      tag.T         `json:"#m,omitempty"`
	Tn      tag.T         `json:"#n,omitempty"`
	To      tag.T         `json:"#o,omitempty"`
	Tp      tag.T         `json:"#p,omitempty"`
	Tq      tag.T         `json:"#q,omitempty"`
	Tr      tag.T         `json:"#r,omitempty"`
	Ts      tag.T         `json:"#s,omitempty"`
	Tt      tag.T         `json:"#t,omitempty"`
	Tu      tag.T         `json:"#u,omitempty"`
	Tv      tag.T         `json:"#v,omitempty"`
	Tw      tag.T         `json:"#w,omitempty"`
	Tx      tag.T         `json:"#x,omitempty"`
	Ty      tag.T         `json:"#y,omitempty"`
	TA      tag.T         `json:"#A,omitempty"`
	TB      tag.T         `json:"#B,omitempty"`
	TC      tag.T         `json:"#C,omitempty"`
	TD      tag.T         `json:"#D,omitempty"`
	TE      tag.T         `json:"#E,omitempty"`
	TF      tag.T         `json:"#F,omitempty"`
	TG      tag.T         `json:"#G,omitempty"`
	TH      tag.T         `json:"#H,omitempty"`
	TI      tag.T         `json:"#I,omitempty"`
	TJ      tag.T         `json:"#J,omitempty"`
	TK      tag.T         `json:"#K,omitempty"`
	TL      tag.T         `json:"#L,omitempty"`
	TM      tag.T         `json:"#M,omitempty"`
	TN      tag.T         `json:"#N,omitempty"`
	TO      tag.T         `json:"#O,omitempty"`
	TP      tag.T         `json:"#P,omitempty"`
	TQ      tag.T         `json:"#Q,omitempty"`
	TR      tag.T         `json:"#R,omitempty"`
	TS      tag.T         `json:"#S,omitempty"`
	TT      tag.T         `json:"#T,omitempty"`
	TU      tag.T         `json:"#U,omitempty"`
	TV      tag.T         `json:"#V,omitempty"`
	TW      tag.T         `json:"#W,omitempty"`
	TX      tag.T         `json:"#X,omitempty"`
	TY      tag.T         `json:"#Y,omitempty"`
	Since   *timestamp.Tp `json:"since,omitempty"`
	Until   *timestamp.Tp `json:"until,omitempty"`
	Limit   int           `json:"limit,omitempty"`
	Search  string        `json:"search,omitempty"`
}

func CopyUnmarshalFilterToFilter(uf *UnmarshalingFilter, f *T) (err error) {
	if uf == nil {
		return fmt.Errorf("cannot copy from nil UnmarshalingFilter")
	}
	if f == nil {
		return fmt.Errorf("cannot copy to nil T")
	}
	// All the easy ones that don't need a generator to handle.
	f.IDs = uf.IDs
	f.Kinds = uf.Kinds
	f.Authors = uf.Authors
	f.Since = uf.Since
	f.Until = uf.Until
	f.Limit = uf.Limit
	f.Search = uf.Search
	f.Tags = make(TagMap)
	// now to populate the map
	if len(uf.Ta) > 0 {
		f.Tags["#a"] = uf.Ta
	}
	if len(uf.Tb) > 0 {
		f.Tags["#b"] = uf.Tb
	}
	if len(uf.Tc) > 0 {
		f.Tags["#c"] = uf.Tc
	}
	if len(uf.Td) > 0 {
		f.Tags["#d"] = uf.Td
	}
	if len(uf.Te) > 0 {
		f.Tags["#e"] = uf.Te
	}
	if len(uf.Tf) > 0 {
		f.Tags["#f"] = uf.Tf
	}
	if len(uf.Tg) > 0 {
		f.Tags["#g"] = uf.Tg
	}
	if len(uf.Th) > 0 {
		f.Tags["#h"] = uf.Th
	}
	if len(uf.Ti) > 0 {
		f.Tags["#i"] = uf.Ti
	}
	if len(uf.Tj) > 0 {
		f.Tags["#j"] = uf.Tj
	}
	if len(uf.Tk) > 0 {
		f.Tags["#k"] = uf.Tk
	}
	if len(uf.Tl) > 0 {
		f.Tags["#l"] = uf.Tl
	}
	if len(uf.Tm) > 0 {
		f.Tags["#m"] = uf.Tm
	}
	if len(uf.Tn) > 0 {
		f.Tags["#n"] = uf.Tn
	}
	if len(uf.To) > 0 {
		f.Tags["#o"] = uf.To
	}
	if len(uf.Tp) > 0 {
		f.Tags["#p"] = uf.Tp
	}
	if len(uf.Tq) > 0 {
		f.Tags["#q"] = uf.Tq
	}
	if len(uf.Tr) > 0 {
		f.Tags["#r"] = uf.Tr
	}
	if len(uf.Ts) > 0 {
		f.Tags["#s"] = uf.Ts
	}
	if len(uf.Tt) > 0 {
		f.Tags["#t"] = uf.Tt
	}
	if len(uf.Tu) > 0 {
		f.Tags["#u"] = uf.Tu
	}
	if len(uf.Tv) > 0 {
		f.Tags["#v"] = uf.Tv
	}
	if len(uf.Tw) > 0 {
		f.Tags["#w"] = uf.Tw
	}
	if len(uf.Tx) > 0 {
		f.Tags["#x"] = uf.Tx
	}
	if len(uf.Ty) > 0 {
		f.Tags["#y"] = uf.Ty
	}
	if len(uf.TA) > 0 {
		f.Tags["#A"] = uf.TA
	}
	if len(uf.TB) > 0 {
		f.Tags["#B"] = uf.TB
	}
	if len(uf.TC) > 0 {
		f.Tags["#C"] = uf.TC
	}
	if len(uf.TD) > 0 {
		f.Tags["#D"] = uf.TD
	}
	if len(uf.TE) > 0 {
		f.Tags["#E"] = uf.TE
	}
	if len(uf.TF) > 0 {
		f.Tags["#F"] = uf.TF
	}
	if len(uf.TG) > 0 {
		f.Tags["#G"] = uf.TG
	}
	if len(uf.TH) > 0 {
		f.Tags["#H"] = uf.TH
	}
	if len(uf.TI) > 0 {
		f.Tags["#I"] = uf.TI
	}
	if len(uf.TJ) > 0 {
		f.Tags["#J"] = uf.TJ
	}
	if len(uf.TK) > 0 {
		f.Tags["#K"] = uf.TK
	}
	if len(uf.TL) > 0 {
		f.Tags["#L"] = uf.TL
	}
	if len(uf.TM) > 0 {
		f.Tags["#M"] = uf.TM
	}
	if len(uf.TN) > 0 {
		f.Tags["#N"] = uf.TN
	}
	if len(uf.TO) > 0 {
		f.Tags["#O"] = uf.TO
	}
	if len(uf.TP) > 0 {
		f.Tags["#P"] = uf.TP
	}
	if len(uf.TQ) > 0 {
		f.Tags["#Q"] = uf.TQ
	}
	if len(uf.TR) > 0 {
		f.Tags["#R"] = uf.TR
	}
	if len(uf.TS) > 0 {
		f.Tags["#S"] = uf.TS
	}
	if len(uf.TT) > 0 {
		f.Tags["#T"] = uf.TT
	}
	if len(uf.TU) > 0 {
		f.Tags["#U"] = uf.TU
	}
	if len(uf.TV) > 0 {
		f.Tags["#V"] = uf.TV
	}
	if len(uf.TW) > 0 {
		f.Tags["#W"] = uf.TW
	}
	if len(uf.TX) > 0 {
		f.Tags["#X"] = uf.TX
	}
	if len(uf.TY) > 0 {
		f.Tags["#Y"] = uf.TY
	}

	return
}
