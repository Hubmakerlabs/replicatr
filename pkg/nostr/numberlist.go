package nostr

type NList []int

// HasNumber returns true if the list contains a given number
func (nl NList) HasNumber(n int) (idx int, has bool) {
	for idx = range nl {
		if nl[idx] == n {
			has = true
			return
		}
	}
	return
}
