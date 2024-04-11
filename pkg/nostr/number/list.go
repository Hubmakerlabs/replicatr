package number

type List []int

// HasNumber returns true if the list contains a given number
func (l List) HasNumber(n int) (idx int, has bool) {
	for idx = range l {
		if l[idx] == n {
			has = true
			return
		}
	}
	return
}
