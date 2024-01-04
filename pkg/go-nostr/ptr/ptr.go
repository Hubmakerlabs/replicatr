package ptr

func Ptr[S any](s S) *S { return &s }
