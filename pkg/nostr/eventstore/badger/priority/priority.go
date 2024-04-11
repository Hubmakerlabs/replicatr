package priority

import "github.com/Hubmakerlabs/replicatr/pkg/nostr/event"

type QueryEvent struct {
	*event.T
	Ser   []byte
	Query int
}

type Queue []*QueryEvent

func (pq Queue) Len() int { return len(pq) }

func (pq Queue) Less(i, j int) bool {
	return pq[i].CreatedAt > pq[j].CreatedAt
}

func (pq Queue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *Queue) Push(x any) {
	item := x.(*QueryEvent)
	*pq = append(*pq, item)
}

func (pq *Queue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // avoid memory leak
	*pq = old[0 : n-1]
	return item
}
