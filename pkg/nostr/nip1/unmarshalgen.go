//go:build ignore

package main

import (
	"fmt"
	"os"
)

func main() {
	f := new(mangle.Buffer)
	fmt.Fprint(f, `package nip1

//go:generate go run unmarshalgen.go

import (
	"fmt"
	"mleku.online/git/replicatr/pkg/nostr/kind"
	"mleku.online/git/replicatr/pkg/nostr/tag"
	"mleku.online/git/replicatr/pkg/nostr/timestamp"
)

type UnmarshalingFilter struct {
	IDs     tag.T         `+"`json:\"ids,omitempty\"`"+`
	Kinds   kind.Array    `+"`json:\"kinds,omitempty\"`"+`
	Authors tag.T         `+"`json:\"authors,omitempty\"`"+`
`)
	for i := 'a'; i < 'z'; i++ {
		fmt.Fprintf(f, "\tT%c      tag.T         `json:\"#%c,omitempty\"`\n", i,
			i)
	}
	for i := 'A'; i < 'Z'; i++ {
		fmt.Fprintf(f, "\tT%c      tag.T         `json:\"#%c,omitempty\"`\n", i,
			i)
	}
	fmt.Fprint(f,
		`	Since   *timestamp.Tp `+"`json:\"since,omitempty\"`"+`
	Until   *timestamp.Tp `+"`json:\"until,omitempty\"`"+`
	Limit   int           `+"`json:\"limit,omitempty\"`"+`
	Search  string        `+"`json:\"search,omitempty\"`"+`
}

func CopyUnmarshalFilterToFilter(uf *UnmarshalingFilter, f *Filter) (e error) {
	if uf == nil {
		return fmt.Errorf("cannot copy from nil UnmarshalingFilter")
	}
	if f == nil {
		return fmt.Errorf("cannot copy to nil Filter")
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
`)
	fmtString := `	if len(uf.T%c) > 0 {
		f.Tags["#%c"] = uf.T%c
	}
`
	for i := 'a'; i < 'z'; i++ {
		fmt.Fprintf(f, fmtString, i, i, i)
	}
	for i := 'A'; i < 'Z'; i++ {
		fmt.Fprintf(f, fmtString, i, i, i)
	}
	fmt.Fprint(f, `
	return
}
`)
	e := os.WriteFile("unmarshalfilter.go", f.Bytes(), 0600)
	if e != nil {
		panic(e)
	}
}
