// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package p2p

import (
	"errors"
	"fmt"
)

const (
	ErrInvalidMsgCode = iota
	ErrInvalidMsg
)

var ErrorToString = map[int]string{
	ErrInvalidMsgCode: "invalid message code",
	ErrInvalidMsg:     "invalid message",
}

// PeerError represents an error that occurred during peer communication
type PeerError struct {
	Code    int
	Message string
}

func newPeerError(code int, format string, v ...interface{}) *PeerError {
	desc, ok := ErrorToString[code]
	if !ok {
		panic("invalid error code")
	}
	err := &PeerError{Code: code, Message: desc}
	if format != "" {
		err.Message += ": " + fmt.Sprintf(format, v...)
	}
	return err
}

func (e *PeerError) Error() string {
	return e.Message
}

var ErrProtocolReturned = errors.New("protocol returned")

var ErrAddPairPeer = errors.New("add a pair peer")

type DiscReason uint

const (
	DiscRequested DiscReason = iota
	DiscNetworkError
	DiscProtocolError
	DiscUselessPeer
	DiscTooManyPeers
	DiscAlreadyConnected
	DiscIncompatibleVersion
	DiscInvalidIdentity
	DiscQuitting
	DiscUnexpectedIdentity
	DiscSelf
	DiscReadTimeout
	DiscPairPeerStop
	DiscSubprotocolError = 0x10
)

var DiscReasonToString = [...]string{
	DiscRequested:           "disconnect requested",
	DiscNetworkError:        "network error",
	DiscProtocolError:       "breach of protocol",
	DiscUselessPeer:         "useless peer",
	DiscTooManyPeers:        "too many peers",
	DiscAlreadyConnected:    "already connected",
	DiscIncompatibleVersion: "incompatible p2p protocol version",
	DiscInvalidIdentity:     "invalid node identity",
	DiscQuitting:            "client quitting",
	DiscUnexpectedIdentity:  "unexpected identity",
	DiscSelf:                "connected to self",
	DiscReadTimeout:         "read timeout",
	DiscPairPeerStop:        "pair peer connection stop",
	DiscSubprotocolError:    "subprotocol error",
}

func (d DiscReason) String() string {
	if len(DiscReasonToString) <= int(d) || int(d) < 0 {
		return fmt.Sprintf("unknown disconnect reason %d", d)
	}
	return DiscReasonToString[int(d)]
}

func (d DiscReason) Error() string {
	return d.String()
}

func discReasonForError(err error) DiscReason {
	if reason, ok := err.(DiscReason); ok {
		return reason
	}
	if err == ErrProtocolReturned {
		return DiscQuitting
	}
	peerError, ok := err.(*PeerError)
	if ok {
		switch peerError.Code {
		case ErrInvalidMsgCode, ErrInvalidMsg:
			return DiscProtocolError
		default:
			return DiscSubprotocolError
		}
	}
	return DiscSubprotocolError
}
