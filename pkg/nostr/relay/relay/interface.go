package relay

type I interface {
	IsConnected() bool
	Write(msg []byte) <-chan error
	Delete(key string)
	URL() string
}
