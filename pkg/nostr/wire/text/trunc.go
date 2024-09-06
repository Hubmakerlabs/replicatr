package text

func Trunc(s string) string {
	if len(s) > 120 {
		return s[:120]
	}
	return s
}
