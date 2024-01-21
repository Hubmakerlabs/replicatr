package types

type Relay interface {
	IsConnected() bool
	Write(msg []byte) <-chan error
	Delete(key string)
	URL() string
}
