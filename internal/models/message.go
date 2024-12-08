package models

import "github.com/gotd/td/tg"

// MessageTask represents a message processing task with all required data
type MessageTask struct {
	Message     *tg.Message
	PeerChannel *tg.PeerChannel
	Channel     *tg.InputPeerChannel
	Attempt     int
}
