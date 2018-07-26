package message

type PeerId string

func (o PeerId) Abbr(length int) string {
	return string(o[:length])
}
