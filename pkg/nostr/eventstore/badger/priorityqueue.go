package badger

type Queries []*queryEvent

type PriorityQueue struct {
	Queries
}

func NewPriorityQueue(l int) *PriorityQueue {
	return &PriorityQueue{make([]*queryEvent, 0, l)}
}

func (pq *PriorityQueue) Len() int { return len(pq.Queries) }

// Less returns whether event i is newer (greater) than event j, reverse
// chronological order, the first result will be the newest.
func (pq *PriorityQueue) Less(i, j int) bool {
	return pq.Queries[i].CreatedAt > pq.Queries[j].CreatedAt
}

func (pq *PriorityQueue) Swap(i, j int) {
	pq.Queries[i], pq.Queries[j] = pq.Queries[j], pq.Queries[i]
}

func (pq *PriorityQueue) Push(x any) {
	item := x.(*queryEvent)
	pq.Queries = append(pq.Queries, item)
}

func (pq *PriorityQueue) Pop() any {
	old := pq.Queries
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // avoid memory leak
	pq.Queries = old[0 : n-1]
	return item
}
