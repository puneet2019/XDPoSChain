// Copyright 2015 The go-ethereum Authors
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
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"io"
	mrand "math/rand"
	"net"
	"sync"
	"time"

	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/crypto/ecies"
	"github.com/XinFinOrg/XDPoSChain/metrics"
	"github.com/XinFinOrg/XDPoSChain/p2p/discover"
	"github.com/XinFinOrg/XDPoSChain/rlp"
	"github.com/golang/snappy"
	"golang.org/x/crypto/sha3"
)

const (
	MaxUint24 = ^uint32(0) >> 8

	SSKLen = 16                     // ecies.MaxSharedKeyLength(pubKey) / 2
	SigLen = crypto.SignatureLength // elliptic S256
	PubLen = 64                     // 512 bit pubkey in uncompressed representation without format byte
	ShaLen = 32                     // hash length (for nonce etc)

	AuthMsgLen  = SigLen + ShaLen + PubLen + ShaLen + 1
	AuthRespLen = PubLen + ShaLen + 1

	ECIESOverhead = 65 /* pubkey */ + 16 /* IV */ + 32 /* MAC */

	EncAuthMsgLen  = AuthMsgLen + ECIESOverhead  // size of encrypted pre-EIP-8 initiator handshake
	EncAuthRespLen = AuthRespLen + ECIESOverhead // size of encrypted pre-EIP-8 handshake reply

	// total timeout for encryption handshake and protocol
	// handshake in both directions.
	HandshakeTimeout = 5 * time.Second

	// This is the timeout for sending the disconnect reason.
	// This is shorter than the usual timeout because we don't want
	// to wait if the connection is known to be bad anyway.
	DiscWriteTimeout = 1 * time.Second
)

// ErrPlainMessageTooLarge is returned if a decompressed message length exceeds
// the allowed 24 bits (i.e. length >= 16MB).
var ErrPlainMessageTooLarge = errors.New("message length >= 16MB")

// Rlpx is the transport protocol used by actual (non-test) connections.
// It wraps the frame encoder with locks and read/write deadlines.
type Rlpx struct {
	Fd net.Conn

	Rmu, Wmu sync.Mutex
	Rw       *RlpxFrameRW
}

func newRLPX(fd net.Conn) Transport {
	fd.SetDeadline(time.Now().Add(HandshakeTimeout))
	return &Rlpx{Fd: fd}
}

func (t *Rlpx) ReadMsg() (Msg, error) {
	t.Rmu.Lock()
	defer t.Rmu.Unlock()
	t.Fd.SetReadDeadline(time.Now().Add(FrameReadTimeout))
	return t.Rw.ReadMsg()
}

func (t *Rlpx) WriteMsg(msg Msg) error {
	t.Wmu.Lock()
	defer t.Wmu.Unlock()
	t.Fd.SetWriteDeadline(time.Now().Add(FrameWriteTimeout))
	return t.Rw.WriteMsg(msg)
}

func (t *Rlpx) Close(err error) {
	t.Wmu.Lock()
	defer t.Wmu.Unlock()
	// Tell the remote end why we're disconnecting if possible.
	if t.Rw != nil {
		if r, ok := err.(DiscReason); ok && r != DiscNetworkError {
			// rlpx tries to send DiscReason to disconnected peer
			// if the connection is net.Pipe (in-memory simulation)
			// it hangs forever, since net.Pipe does not implement
			// a write deadline. Because of this only try to send
			// the disconnect reason message if there is no error.
			if err := t.Fd.SetWriteDeadline(time.Now().Add(DiscWriteTimeout)); err == nil {
				SendItems(t.Rw, DiscMsg, r)
			}
		}
	}
	t.Fd.Close()
}

func (t *Rlpx) DoProtoHandshake(our *ProtoHandshake) (their *ProtoHandshake, err error) {

	// Writing our handshake happens concurrently, we prefer
	// returning the handshake read error. If the remote side
	// disconnects us early with a valid reason, we should return it
	// as the error so it can be tracked elsewhere.
	werr := make(chan error, 1)
	go func() { werr <- Send(t.Rw, HandshakeMsg, our) }()
	if their, err = readProtocolHandshake(t.Rw, our); err != nil {
		<-werr // make sure the write terminates too
		return nil, err
	}
	if err := <-werr; err != nil {
		return nil, fmt.Errorf("write error: %v", err)
	}
	// If the protocol version supports Snappy encoding, upgrade immediately
	t.Rw.Snappy = their.Version >= SnappyProtocolVersion

	return their, nil
}

func readProtocolHandshake(rw MsgReader, our *ProtoHandshake) (*ProtoHandshake, error) {
	msg, err := rw.ReadMsg()
	if err != nil {
		return nil, err
	}
	if msg.Size > BaseProtocolMaxMsgSize {
		return nil, errors.New("message too big")
	}
	if msg.Code == DiscMsg {
		// Disconnect before protocol handshake is valid according to the
		// spec and we send it ourself if the posthanshake checks fail.
		// We can't return the reason directly, though, because it is echoed
		// back otherwise. Wrap it in a string instead.
		var reason [1]DiscReason
		rlp.Decode(msg.Payload, &reason)
		return nil, reason[0]
	}
	if msg.Code != HandshakeMsg {
		return nil, fmt.Errorf("expected handshake, got %x", msg.Code)
	}
	var hs ProtoHandshake
	if err := msg.Decode(&hs); err != nil {
		return nil, err
	}
	if (hs.ID == discover.NodeID{}) {
		return nil, DiscInvalidIdentity
	}
	return &hs, nil
}

// DoEncHandshake runs the protocol handshake using authenticated
// messages. the protocol handshake is the first authenticated message
// and also verifies whether the encryption handshake 'worked' and the
// remote side actually provided the right public key.
func (t *Rlpx) DoEncHandshake(prv *ecdsa.PrivateKey, dial *discover.Node) (discover.NodeID, error) {
	var (
		sec Secrets
		err error
	)
	if dial == nil {
		sec, err = ReceiverEncHandshake(t.Fd, prv, nil)
	} else {
		sec, err = InitiatorEncHandshake(t.Fd, prv, dial.ID)
	}
	if err != nil {
		return discover.NodeID{}, err
	}
	t.Wmu.Lock()
	t.Rw = newRLPXFrameRW(t.Fd, sec)
	t.Wmu.Unlock()
	return sec.RemoteID, nil
}

// EncHandshake contains the state of the encryption handshake.
type EncHandshake struct {
	Initiator bool
	RemoteID  discover.NodeID

	RemotePub            *ecies.PublicKey  // remote-pubk
	InitNonce, RespNonce []byte            // nonce
	RandomPrivKey        *ecies.PrivateKey // ecdhe-random
	RemoteRandomPub      *ecies.PublicKey  // ecdhe-random-pubk
}

// Secrets represents the connection secrets
// which are negotiated during the encryption handshake.
type Secrets struct {
	RemoteID              discover.NodeID
	AES, MAC              []byte
	EgressMAC, IngressMAC hash.Hash
	Token                 []byte
}

// RLPx v4 handshake auth (defined in EIP-8).
type AuthMsgV4 struct {
	GotPlain bool // whether read packet had plain format.

	Signature       [SigLen]byte
	InitiatorPubkey [PubLen]byte
	Nonce           [ShaLen]byte
	Version         uint

	// Ignore additional fields (forward-compatibility)
	Rest []rlp.RawValue `rlp:"tail"`
}

// RLPx v4 handshake response (defined in EIP-8).
type AuthRespV4 struct {
	RandomPubkey [PubLen]byte
	Nonce        [ShaLen]byte
	Version      uint

	// Ignore additional fields (forward-compatibility)
	Rest []rlp.RawValue `rlp:"tail"`
}

// Secrets is called after the handshake is completed.
// It extracts the connection secrets from the handshake values.
func (h *EncHandshake) Secrets(auth, authResp []byte) (Secrets, error) {
	ecdheSecret, err := h.RandomPrivKey.GenerateShared(h.RemoteRandomPub, SSKLen, SSKLen)
	if err != nil {
		return Secrets{}, err
	}

	// derive base secrets from ephemeral key agreement
	sharedSecret := crypto.Keccak256(ecdheSecret, crypto.Keccak256(h.RespNonce, h.InitNonce))
	aesSecret := crypto.Keccak256(ecdheSecret, sharedSecret)
	s := Secrets{
		RemoteID: h.RemoteID,
		AES:      aesSecret,
		MAC:      crypto.Keccak256(ecdheSecret, aesSecret),
	}

	// setup sha3 instances for the MACs
	mac1 := sha3.NewLegacyKeccak256()
	mac1.Write(xor(s.MAC, h.RespNonce))
	mac1.Write(auth)
	mac2 := sha3.NewLegacyKeccak256()
	mac2.Write(xor(s.MAC, h.InitNonce))
	mac2.Write(authResp)
	if h.Initiator {
		s.EgressMAC, s.IngressMAC = mac1, mac2
	} else {
		s.EgressMAC, s.IngressMAC = mac2, mac1
	}

	return s, nil
}

// StaticSharedSecret returns the static shared secret, the result
// of key agreement between the local and remote static node key.
func (h *EncHandshake) StaticSharedSecret(prv *ecdsa.PrivateKey) ([]byte, error) {
	return ecies.ImportECDSA(prv).GenerateShared(h.RemotePub, SSKLen, SSKLen)
}

// InitiatorEncHandshake negotiates a session token on conn.
// it should be called on the dialing side of the connection.
//
// prv is the local client's private key.
func InitiatorEncHandshake(conn io.ReadWriter, prv *ecdsa.PrivateKey, remoteID discover.NodeID) (s Secrets, err error) {
	h := &EncHandshake{Initiator: true, RemoteID: remoteID}
	authMsg, err := h.makeAuthMsg(prv)
	if err != nil {
		return s, err
	}
	authPacket, err := sealEIP8(authMsg, h)
	if err != nil {
		return s, err
	}
	if _, err = conn.Write(authPacket); err != nil {
		return s, err
	}

	authRespMsg := new(AuthRespV4)
	authRespPacket, err := readHandshakeMsg(authRespMsg, EncAuthRespLen, prv, conn)
	if err != nil {
		return s, err
	}
	if err := h.handleAuthResp(authRespMsg); err != nil {
		return s, err
	}
	return h.Secrets(authPacket, authRespPacket)
}

// makeAuthMsg creates the initiator handshake message.
func (h *EncHandshake) makeAuthMsg(prv *ecdsa.PrivateKey) (*AuthMsgV4, error) {
	rpub, err := h.RemoteID.Pubkey()
	if err != nil {
		return nil, fmt.Errorf("bad remoteID: %v", err)
	}
	h.RemotePub = ecies.ImportECDSAPublic(rpub)
	// Generate random initiator nonce.
	h.InitNonce = make([]byte, ShaLen)
	if _, err := rand.Read(h.InitNonce); err != nil {
		return nil, err
	}
	// Generate random keypair to for ECDH.
	h.RandomPrivKey, err = ecies.GenerateKey(rand.Reader, crypto.S256(), nil)
	if err != nil {
		return nil, err
	}

	// Sign known message: static-shared-secret ^ nonce
	token, err := h.StaticSharedSecret(prv)
	if err != nil {
		return nil, err
	}
	signed := xor(token, h.InitNonce)
	signature, err := crypto.Sign(signed, h.RandomPrivKey.ExportECDSA())
	if err != nil {
		return nil, err
	}

	msg := new(AuthMsgV4)
	copy(msg.Signature[:], signature)
	copy(msg.InitiatorPubkey[:], crypto.FromECDSAPub(&prv.PublicKey)[1:])
	copy(msg.Nonce[:], h.InitNonce)
	msg.Version = 4
	return msg, nil
}

func (h *EncHandshake) handleAuthResp(msg *AuthRespV4) (err error) {
	h.RespNonce = msg.Nonce[:]
	h.RemoteRandomPub, err = ImportPublicKey(msg.RandomPubkey[:])
	return err
}

// ReceiverEncHandshake negotiates a session token on conn.
// it should be called on the listening side of the connection.
//
// prv is the local client's private key.
// token is the token from a previous session with this node.
func ReceiverEncHandshake(conn io.ReadWriter, prv *ecdsa.PrivateKey, token []byte) (s Secrets, err error) {
	authMsg := new(AuthMsgV4)
	authPacket, err := readHandshakeMsg(authMsg, EncAuthMsgLen, prv, conn)
	if err != nil {
		return s, err
	}
	h := new(EncHandshake)
	if err := h.handleAuthMsg(authMsg, prv); err != nil {
		return s, err
	}

	authRespMsg, err := h.makeAuthResp()
	if err != nil {
		return s, err
	}
	var authRespPacket []byte
	if authMsg.GotPlain {
		authRespPacket, err = authRespMsg.sealPlain(h)
	} else {
		authRespPacket, err = sealEIP8(authRespMsg, h)
	}
	if err != nil {
		return s, err
	}
	if _, err = conn.Write(authRespPacket); err != nil {
		return s, err
	}
	return h.Secrets(authPacket, authRespPacket)
}

func (h *EncHandshake) handleAuthMsg(msg *AuthMsgV4, prv *ecdsa.PrivateKey) error {
	// Import the remote identity.
	h.InitNonce = msg.Nonce[:]
	h.RemoteID = msg.InitiatorPubkey
	rpub, err := h.RemoteID.Pubkey()
	if err != nil {
		return fmt.Errorf("bad remoteID: %#v", err)
	}
	h.RemotePub = ecies.ImportECDSAPublic(rpub)

	// Generate random keypair for ECDH.
	// If a private key is already set, use it instead of generating one (for testing).
	if h.RandomPrivKey == nil {
		h.RandomPrivKey, err = ecies.GenerateKey(rand.Reader, crypto.S256(), nil)
		if err != nil {
			return err
		}
	}

	// Check the signature.
	token, err := h.StaticSharedSecret(prv)
	if err != nil {
		return err
	}
	signedMsg := xor(token, h.InitNonce)
	remoteRandomPub, err := crypto.Ecrecover(signedMsg, msg.Signature[:])
	if err != nil {
		return err
	}
	h.RemoteRandomPub, _ = ImportPublicKey(remoteRandomPub)
	return nil
}

func (h *EncHandshake) makeAuthResp() (msg *AuthRespV4, err error) {
	// Generate random nonce.
	h.RespNonce = make([]byte, ShaLen)
	if _, err = rand.Read(h.RespNonce); err != nil {
		return nil, err
	}

	msg = new(AuthRespV4)
	copy(msg.Nonce[:], h.RespNonce)
	copy(msg.RandomPubkey[:], exportPubkey(&h.RandomPrivKey.PublicKey))
	msg.Version = 4
	return msg, nil
}

func (msg *AuthMsgV4) sealPlain(h *EncHandshake) ([]byte, error) {
	buf := make([]byte, AuthMsgLen)
	n := copy(buf, msg.Signature[:])
	n += copy(buf[n:], crypto.Keccak256(exportPubkey(&h.RandomPrivKey.PublicKey)))
	n += copy(buf[n:], msg.InitiatorPubkey[:])
	n += copy(buf[n:], msg.Nonce[:])
	buf[n] = 0 // token-flag
	return ecies.Encrypt(rand.Reader, h.RemotePub, buf, nil, nil)
}

func (msg *AuthMsgV4) DecodePlain(input []byte) {
	n := copy(msg.Signature[:], input)
	n += ShaLen // skip sha3(initiator-ephemeral-pubk)
	n += copy(msg.InitiatorPubkey[:], input[n:])
	copy(msg.Nonce[:], input[n:])
	msg.Version = 4
	msg.GotPlain = true
}

func (msg *AuthRespV4) sealPlain(hs *EncHandshake) ([]byte, error) {
	buf := make([]byte, AuthRespLen)
	n := copy(buf, msg.RandomPubkey[:])
	copy(buf[n:], msg.Nonce[:])
	return ecies.Encrypt(rand.Reader, hs.RemotePub, buf, nil, nil)
}

func (msg *AuthRespV4) DecodePlain(input []byte) {
	n := copy(msg.RandomPubkey[:], input)
	copy(msg.Nonce[:], input[n:])
	msg.Version = 4
}

var PadSpace = make([]byte, 300)

func sealEIP8(msg interface{}, h *EncHandshake) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := rlp.Encode(buf, msg); err != nil {
		return nil, err
	}
	// pad with random amount of data. the amount needs to be at least 100 bytes to make
	// the message distinguishable from pre-EIP-8 handshakes.
	pad := PadSpace[:mrand.Intn(len(PadSpace)-100)+100]
	buf.Write(pad)
	prefix := make([]byte, 2)
	binary.BigEndian.PutUint16(prefix, uint16(buf.Len()+ECIESOverhead))

	enc, err := ecies.Encrypt(rand.Reader, h.RemotePub, buf.Bytes(), nil, prefix)
	return append(prefix, enc...), err
}

type PlainDecoder interface {
	DecodePlain([]byte)
}

func readHandshakeMsg(msg PlainDecoder, plainSize int, prv *ecdsa.PrivateKey, r io.Reader) ([]byte, error) {
	buf := make([]byte, plainSize)
	if _, err := io.ReadFull(r, buf); err != nil {
		return buf, err
	}
	// Attempt decoding pre-EIP-8 "plain" format.
	key := ecies.ImportECDSA(prv)
	if dec, err := key.Decrypt(buf, nil, nil); err == nil {
		msg.DecodePlain(dec)
		return buf, nil
	}
	// Could be EIP-8 format, try that.
	prefix := buf[:2]
	size := binary.BigEndian.Uint16(prefix)
	if size < uint16(plainSize) {
		return buf, fmt.Errorf("size underflow, need at least %d bytes", plainSize)
	}
	buf = append(buf, make([]byte, size-uint16(plainSize)+2)...)
	if _, err := io.ReadFull(r, buf[plainSize:]); err != nil {
		return buf, err
	}
	dec, err := key.Decrypt(buf[2:], nil, prefix)
	if err != nil {
		return buf, err
	}
	// Can't use rlp.DecodeBytes here because it rejects
	// trailing data (forward-compatibility).
	s := rlp.NewStream(bytes.NewReader(dec), 0)
	return buf, s.Decode(msg)
}

// ImportPublicKey unmarshals 512 bit public keys.
func ImportPublicKey(pubKey []byte) (*ecies.PublicKey, error) {
	var pubKey65 []byte
	switch len(pubKey) {
	case 64:
		// add 'uncompressed key' flag
		pubKey65 = append([]byte{0x04}, pubKey...)
	case 65:
		pubKey65 = pubKey
	default:
		return nil, fmt.Errorf("invalid public key length %v (expect 64/65)", len(pubKey))
	}
	// TODO: fewer pointless conversions
	pub, err := crypto.UnmarshalPubkey(pubKey65)
	if err != nil {
		return nil, err
	}
	return ecies.ImportECDSAPublic(pub), nil
}

func exportPubkey(pub *ecies.PublicKey) []byte {
	if pub == nil {
		panic("nil pubkey")
	}
	return elliptic.Marshal(pub.Curve, pub.X, pub.Y)[1:]
}

func xor(one, other []byte) (xor []byte) {
	xor = make([]byte, len(one))
	for i := 0; i < len(one); i++ {
		xor[i] = one[i] ^ other[i]
	}
	return xor
}

var (
	// this is used in place of actual frame header data.
	// TODO: replace this when Msg contains the protocol type code.
	ZeroHeader = []byte{0xC2, 0x80, 0x80}
	// sixteen zero bytes
	Zero16 = make([]byte, 16)
)

// RlpxFrameRW implements a simplified version of RLPx framing.
// chunked messages are not supported and all headers are equal to
// zeroHeader.
//
// rlpxFrameRW is not safe for concurrent use from multiple goroutines.
type RlpxFrameRW struct {
	Conn io.ReadWriter
	Enc  cipher.Stream
	Dec  cipher.Stream

	MacCipher  cipher.Block
	EgressMAC  hash.Hash
	IngressMAC hash.Hash

	Snappy bool
}

func newRLPXFrameRW(conn io.ReadWriter, s Secrets) *RlpxFrameRW {
	macc, err := aes.NewCipher(s.MAC)
	if err != nil {
		panic("invalid MAC secret: " + err.Error())
	}
	encc, err := aes.NewCipher(s.AES)
	if err != nil {
		panic("invalid AES secret: " + err.Error())
	}
	// we use an all-zeroes IV for AES because the key used
	// for encryption is ephemeral.
	iv := make([]byte, encc.BlockSize())
	return &RlpxFrameRW{
		Conn:       conn,
		Enc:        cipher.NewCTR(encc, iv),
		Dec:        cipher.NewCTR(encc, iv),
		MacCipher:  macc,
		EgressMAC:  s.EgressMAC,
		IngressMAC: s.IngressMAC,
	}
}

func (rw *RlpxFrameRW) WriteMsg(msg Msg) error {
	ptype, _ := rlp.EncodeToBytes(msg.Code)

	// if snappy is enabled, compress message now
	if rw.Snappy {
		if msg.Size > MaxUint24 {
			return ErrPlainMessageTooLarge
		}
		payload, _ := io.ReadAll(msg.Payload)
		payload = snappy.Encode(nil, payload)

		msg.Payload = bytes.NewReader(payload)
		msg.Size = uint32(len(payload))
	}
	msg.MeterSize = msg.Size
	if metrics.Enabled() && msg.MeterCap.Name != "" { // don't meter non-subprotocol messages
		metrics.GetOrRegisterMeter(fmt.Sprintf("%s/%s/%d/%#02x", MetricsOutboundTraffic, msg.MeterCap.Name, msg.MeterCap.Version, msg.MeterCode), nil).Mark(int64(msg.MeterSize))
	}
	// write header
	headbuf := make([]byte, 32)
	fsize := uint32(len(ptype)) + msg.Size
	if fsize > MaxUint24 {
		return errors.New("message size overflows uint24")
	}
	putInt24(fsize, headbuf) // TODO: check overflow
	copy(headbuf[3:], ZeroHeader)
	rw.Enc.XORKeyStream(headbuf[:16], headbuf[:16]) // first half is now encrypted

	// write header MAC
	copy(headbuf[16:], UpdateMAC(rw.EgressMAC, rw.MacCipher, headbuf[:16]))
	if _, err := rw.Conn.Write(headbuf); err != nil {
		return err
	}

	// write encrypted frame, updating the egress MAC hash with
	// the data written to conn.
	tee := cipher.StreamWriter{S: rw.Enc, W: io.MultiWriter(rw.Conn, rw.EgressMAC)}
	if _, err := tee.Write(ptype); err != nil {
		return err
	}
	if _, err := io.Copy(tee, msg.Payload); err != nil {
		return err
	}
	if padding := fsize % 16; padding > 0 {
		if _, err := tee.Write(Zero16[:16-padding]); err != nil {
			return err
		}
	}

	// write frame MAC. egress MAC hash is up to date because
	// frame content was written to it as well.
	fmacseed := rw.EgressMAC.Sum(nil)
	mac := UpdateMAC(rw.EgressMAC, rw.MacCipher, fmacseed)
	_, err := rw.Conn.Write(mac)
	return err
}

func (rw *RlpxFrameRW) ReadMsg() (msg Msg, err error) {
	// read the header
	headbuf := make([]byte, 32)
	if _, err := io.ReadFull(rw.Conn, headbuf); err != nil {
		return msg, err
	}
	// verify header mac
	shouldMAC := UpdateMAC(rw.IngressMAC, rw.MacCipher, headbuf[:16])
	if !hmac.Equal(shouldMAC, headbuf[16:]) {
		return msg, errors.New("bad header MAC")
	}
	rw.Dec.XORKeyStream(headbuf[:16], headbuf[:16]) // first half is now decrypted
	fsize := readInt24(headbuf)
	// ignore protocol type for now

	// read the frame content
	var rsize = fsize // frame size rounded up to 16 byte boundary
	if padding := fsize % 16; padding > 0 {
		rsize += 16 - padding
	}
	framebuf := make([]byte, rsize)
	if _, err := io.ReadFull(rw.Conn, framebuf); err != nil {
		return msg, err
	}

	// read and validate frame MAC. we can re-use headbuf for that.
	rw.IngressMAC.Write(framebuf)
	fmacseed := rw.IngressMAC.Sum(nil)
	if _, err := io.ReadFull(rw.Conn, headbuf[:16]); err != nil {
		return msg, err
	}
	shouldMAC = UpdateMAC(rw.IngressMAC, rw.MacCipher, fmacseed)
	if !hmac.Equal(shouldMAC, headbuf[:16]) {
		return msg, errors.New("bad frame MAC")
	}

	// decrypt frame content
	rw.Dec.XORKeyStream(framebuf, framebuf)

	// decode message code
	content := bytes.NewReader(framebuf[:fsize])
	if err := rlp.Decode(content, &msg.Code); err != nil {
		return msg, err
	}
	msg.Size = uint32(content.Len())
	msg.MeterSize = msg.Size
	msg.Payload = content

	// if snappy is enabled, verify and decompress message
	if rw.Snappy {
		payload, err := io.ReadAll(msg.Payload)
		if err != nil {
			return msg, err
		}
		size, err := snappy.DecodedLen(payload)
		if err != nil {
			return msg, err
		}
		if size > int(MaxUint24) {
			return msg, ErrPlainMessageTooLarge
		}
		payload, err = snappy.Decode(nil, payload)
		if err != nil {
			return msg, err
		}
		msg.Size, msg.Payload = uint32(size), bytes.NewReader(payload)
	}
	return msg, nil
}

// UpdateMAC reseeds the given hash with encrypted seed.
// it returns the first 16 bytes of the hash sum after seeding.
func UpdateMAC(mac hash.Hash, block cipher.Block, seed []byte) []byte {
	aesbuf := make([]byte, aes.BlockSize)
	block.Encrypt(aesbuf, mac.Sum(nil))
	for i := range aesbuf {
		aesbuf[i] ^= seed[i]
	}
	mac.Write(aesbuf)
	return mac.Sum(nil)[:16]
}

func readInt24(b []byte) uint32 {
	return uint32(b[2]) | uint32(b[1])<<8 | uint32(b[0])<<16
}

func putInt24(v uint32, b []byte) {
	b[0] = byte(v >> 16)
	b[1] = byte(v >> 8)
	b[2] = byte(v)
}
