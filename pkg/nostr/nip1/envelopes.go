package nip1

type EventEnvelope struct {
	SubscriptionID string
	Event
}
type ReqEnvelope struct {
	SubscriptionID string
	Filters
}
type CountEnvelope struct {
	SubscriptionID string
	Filters
	Count *int64
}
type NoticeEnvelope string
type EOSEEnvelope string
type CloseEnvelope string
type OKEnvelope struct {
	EventID string
	OK      bool
	Reason  string
}
type AuthEnvelope struct {
	Challenge *string
	Event     Event
}
