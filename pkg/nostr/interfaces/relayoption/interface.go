package relayoption

// I is the type of the argument passed for that. Some examples of this are
// WithNoticeHandler and WithAuthHandler.
type I interface {
	IsRelayOption()
}
