package text

func DefLimit(s string) string {
	if len(s) > 192 {
		return s[:192]
	}
	return s
}
