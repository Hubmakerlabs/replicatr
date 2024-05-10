package relay

type I interface {
	IsConnected() bool
	Write(msg []byte) (ch chan error)
	Delete(key string)
	URL() string
}
