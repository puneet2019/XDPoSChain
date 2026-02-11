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

// Package p2p implements the Ethereum p2p network protocols.
package p2p

import (
	"crypto/ecdsa"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/common/mclock"
	"github.com/XinFinOrg/XDPoSChain/event"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/p2p/discover"
	"github.com/XinFinOrg/XDPoSChain/p2p/discv5"
	"github.com/XinFinOrg/XDPoSChain/p2p/nat"
	"github.com/XinFinOrg/XDPoSChain/p2p/netutil"
)

const (
	DefaultDialTimeout = 15 * time.Second

	// Connectivity defaults.
	MaxActiveDialTasks     = 16
	DefaultMaxPendingPeers = 50
	DefaultDialRatio       = 3

	// Maximum time allowed for reading a complete message.
	// This is effectively the amount of time a connection can be idle.
	FrameReadTimeout = 30 * time.Second

	// Maximum amount of time allowed for writing a complete message.
	FrameWriteTimeout = 20 * time.Second
)

var (
	ErrServerStopped       = errors.New("server stopped")
	ErrEncHandshakeError   = errors.New("rlpx enc error")
	ErrProtoHandshakeError = errors.New("rlpx proto error")
)

// Config holds Server options.
type Config struct {
	// This field must be set to a valid secp256k1 private key.
	PrivateKey *ecdsa.PrivateKey `toml:"-"`

	// MaxPeers is the maximum number of peers that can be
	// connected. It must be greater than zero.
	MaxPeers int

	// MaxPendingPeers is the maximum number of peers that can be pending in the
	// handshake phase, counted separately for inbound and outbound connections.
	// Zero defaults to preset values.
	MaxPendingPeers int `toml:",omitempty"`

	// DialRatio controls the ratio of inbound to dialed connections.
	// Example: a DialRatio of 2 allows 1/2 of connections to be dialed.
	// Setting DialRatio to zero defaults it to 3.
	DialRatio int `toml:",omitempty"`

	// NoDiscovery can be used to disable the peer discovery mechanism.
	// Disabling is useful for protocol debugging (manual topology).
	NoDiscovery bool

	// DiscoveryV5 specifies whether the new topic-discovery based V5 discovery
	// protocol should be started or not.
	DiscoveryV5 bool `toml:",omitempty"`

	// Name sets the node name of this server.
	Name string `toml:"-"`

	// BootstrapNodes are used to establish connectivity
	// with the rest of the network.
	BootstrapNodes []*discover.Node

	// BootstrapNodesV5 are used to establish connectivity
	// with the rest of the network using the V5 discovery
	// protocol.
	BootstrapNodesV5 []*discv5.Node `toml:",omitempty"`

	// Static nodes are used as pre-configured connections which are always
	// maintained and re-connected on disconnects.
	StaticNodes []*discover.Node

	// Trusted nodes are used as pre-configured connections which are always
	// allowed to connect, even above the peer limit.
	TrustedNodes []*discover.Node

	// Connectivity can be restricted to certain IP networks.
	// If this option is set to a non-nil value, only hosts which match one of the
	// IP networks contained in the list are considered.
	NetRestrict *netutil.Netlist `toml:",omitempty"`

	// NodeDatabase is the path to the database containing the previously seen
	// live nodes in the network.
	NodeDatabase string `toml:",omitempty"`

	// Protocols should contain the protocols supported
	// by the server. Matching protocols are launched for
	// each peer.
	Protocols []Protocol `toml:"-"`

	// If ListenAddr is set to a non-nil address, the server
	// will listen for incoming connections.
	//
	// If the port is zero, the operating system will pick a port. The
	// ListenAddr field will be updated with the actual address when
	// the server is started.
	ListenAddr string

	// If set to a non-nil value, the given NAT port mapper
	// is used to make the listening port available to the
	// Internet.
	NAT nat.Interface `toml:",omitempty"`

	// If Dialer is set to a non-nil value, the given Dialer
	// is used to dial outbound peer connections.
	Dialer NodeDialer `toml:"-"`

	// If NoDial is true, the server will not dial any peers.
	NoDial bool `toml:",omitempty"`

	// If EnableMsgEvents is set then the server will emit PeerEvents
	// whenever a message is sent to or received from a peer
	EnableMsgEvents bool

	// Logger is a custom logger to use with the p2p.Server.
	Logger log.Logger `toml:",omitempty"`
}

// Server manages all peer connections.
type Server struct {
	// Config fields may not be modified while the server is running.
	Config

	// Hooks for testing. These are useful because we can inhibit
	// the whole protocol stack.
	NewTransport func(net.Conn) Transport
	NewPeerHook  func(*Peer)

	Lock    sync.Mutex // protects running
	Running bool

	Ntab         DiscoverTable
	Listener     net.Listener
	OurHandshake *ProtoHandshake
	LastLookup   time.Time
	DiscV5       *discv5.Network

	// These are for Peers, PeerCount (and nothing else).
	PeerOp     chan PeerOpFunc
	PeerOpDone chan struct{}

	Quit          chan struct{}
	Addstatic     chan *discover.Node
	Removestatic  chan *discover.Node
	Addtrusted    chan *discover.Node
	Removetrusted chan *discover.Node
	Posthandshake chan *Conn
	Addpeer       chan *Conn
	Delpeer       chan PeerDrop
	LoopWG        sync.WaitGroup // loop, listenLoop
	PeerFeed      event.Feed
	Log           log.Logger
}

type PeerOpFunc func(map[discover.NodeID]*Peer)

type PeerDrop struct {
	*Peer
	Err       error
	Requested bool // true if signaled by the peer
}

type ConnFlag int32

const (
	DynDialedConn ConnFlag = 1 << iota
	StaticDialedConn
	InboundConn
	TrustedConn
)

// Conn wraps a network connection with information gathered
// during the two handshakes.
type Conn struct {
	Fd net.Conn
	Transport
	Flags ConnFlag
	Cont  chan error      // The run loop uses cont to signal errors to SetupConn.
	Id    discover.NodeID // valid after the encryption handshake
	Caps  []Cap           // valid after the protocol handshake
	Name  string          // valid after the protocol handshake
}

type Transport interface {
	// The two handshakes.
	DoEncHandshake(prv *ecdsa.PrivateKey, dialDest *discover.Node) (discover.NodeID, error)
	DoProtoHandshake(our *ProtoHandshake) (*ProtoHandshake, error)
	// The MsgReadWriter can only be used after the encryption
	// handshake has completed. The code uses conn.id to track this
	// by setting it to a non-nil value after the encryption handshake.
	MsgReadWriter
	// transports must provide Close because we use MsgPipe in some of
	// the tests. Closing the actual network connection doesn't do
	// anything in those tests because NsgPipe doesn't use it.
	Close(err error)
}

func (c *Conn) String() string {
	s := c.Flags.String()
	if (c.Id != discover.NodeID{}) {
		s += " " + c.Id.String()
	}
	s += " " + c.Fd.RemoteAddr().String()
	return s
}

func (f ConnFlag) String() string {
	s := ""
	if f&TrustedConn != 0 {
		s += "-trusted"
	}
	if f&DynDialedConn != 0 {
		s += "-dyndial"
	}
	if f&StaticDialedConn != 0 {
		s += "-staticdial"
	}
	if f&InboundConn != 0 {
		s += "-inbound"
	}
	if s != "" {
		s = s[1:]
	}
	return s
}

func (c *Conn) Is(f ConnFlag) bool {
	flags := ConnFlag(atomic.LoadInt32((*int32)(&c.Flags)))
	return flags&f != 0
}

func (c *Conn) Set(f ConnFlag, val bool) {
	flags := ConnFlag(atomic.LoadInt32((*int32)(&c.Flags)))
	if val {
		flags |= f
	} else {
		flags &= ^f
	}
	atomic.StoreInt32((*int32)(&c.Flags), int32(flags))
}

// Peers returns all connected peers.
func (srv *Server) Peers() []*Peer {
	var ps []*Peer
	select {
	// Note: We'd love to put this function into a variable but
	// that seems to cause a weird compiler error in some
	// environments.
	case srv.PeerOp <- func(peers map[discover.NodeID]*Peer) {
		for _, p := range peers {
			ps = append(ps, p)
		}
	}:
		<-srv.PeerOpDone
	case <-srv.Quit:
	}
	return ps
}

// PeerCount returns the number of connected peers.
func (srv *Server) PeerCount() int {
	var count int
	select {
	case srv.PeerOp <- func(ps map[discover.NodeID]*Peer) { count = len(ps) }:
		<-srv.PeerOpDone
	case <-srv.Quit:
	}
	return count
}

// AddPeer connects to the given node and maintains the connection until the
// server is shut down. If the connection fails for any reason, the server will
// attempt to reconnect the peer.
func (srv *Server) AddPeer(node *discover.Node) {

	select {
	case srv.Addstatic <- node:
	case <-srv.Quit:
	}
}

// RemovePeer disconnects from the given node
func (srv *Server) RemovePeer(node *discover.Node) {
	select {
	case srv.Removestatic <- node:
	case <-srv.Quit:
	}
}

// AddTrustedPeer adds the given node to a reserved whitelist which allows the
// node to always connect, even if the slot are full.
func (srv *Server) AddTrustedPeer(node *discover.Node) {
	select {
	case srv.Addtrusted <- node:
	case <-srv.Quit:
	}
}

// RemoveTrustedPeer removes the given node from the trusted peer set.
func (srv *Server) RemoveTrustedPeer(node *discover.Node) {
	select {
	case srv.Removetrusted <- node:
	case <-srv.Quit:
	}
}

// SubscribePeers subscribes the given channel to peer events
func (srv *Server) SubscribeEvents(ch chan *PeerEvent) event.Subscription {
	return srv.PeerFeed.Subscribe(ch)
}

// Self returns the local node's endpoint information.
func (srv *Server) Self() *discover.Node {
	srv.Lock.Lock()
	defer srv.Lock.Unlock()

	if !srv.Running {
		return &discover.Node{IP: net.ParseIP("0.0.0.0")}
	}
	return srv.makeSelf(srv.Listener, srv.Ntab)
}

func (srv *Server) makeSelf(listener net.Listener, ntab DiscoverTable) *discover.Node {
	// If the server's not running, return an empty node.
	// If the node is running but discovery is off, manually assemble the node infos.
	if ntab == nil {
		// Inbound connections disabled, use zero address.
		if listener == nil {
			return &discover.Node{IP: net.ParseIP("0.0.0.0"), ID: discover.PubkeyID(&srv.PrivateKey.PublicKey)}
		}
		// Otherwise inject the listener address too
		addr := listener.Addr().(*net.TCPAddr)
		return &discover.Node{
			ID:  discover.PubkeyID(&srv.PrivateKey.PublicKey),
			IP:  addr.IP,
			TCP: uint16(addr.Port),
		}
	}
	// Otherwise return the discovery node.
	return ntab.Self()
}

// Stop terminates the server and all active peer connections.
// It blocks until all active connections have been closed.
func (srv *Server) Stop() {
	srv.Lock.Lock()
	defer srv.Lock.Unlock()
	if !srv.Running {
		return
	}
	srv.Running = false
	if srv.Listener != nil {
		// this unblocks listener Accept
		srv.Listener.Close()
	}
	close(srv.Quit)
	srv.LoopWG.Wait()
}

// SharedUDPConn implements a shared connection. Write sends messages to the underlying connection while read returns
// messages that were found unprocessable and sent to the unhandled channel by the primary listener.
type SharedUDPConn struct {
	*net.UDPConn
	Unhandled chan discover.ReadPacket
}

// ReadFromUDP implements discv5.conn
func (s *SharedUDPConn) ReadFromUDP(b []byte) (n int, addr *net.UDPAddr, err error) {
	packet, ok := <-s.Unhandled
	if !ok {
		return 0, nil, errors.New("connection was closed")
	}
	l := len(packet.Data)
	if l > len(b) {
		l = len(b)
	}
	copy(b[:l], packet.Data[:l])
	return l, packet.Addr, nil
}

// Close implements discv5.conn
func (s *SharedUDPConn) Close() error {
	return nil
}

// Start starts running the server.
// Servers can not be re-used after stopping.
func (srv *Server) Start() (err error) {
	srv.Lock.Lock()
	defer srv.Lock.Unlock()
	if srv.Running {
		return errors.New("server already running")
	}
	srv.Running = true
	srv.Log = srv.Config.Logger
	if srv.Log == nil {
		srv.Log = log.New()
	}
	srv.Log.Info("Starting P2P networking")

	// static fields
	if srv.PrivateKey == nil {
		return errors.New("Server.PrivateKey must be set to a non-nil key")
	}
	if srv.NewTransport == nil {
		srv.NewTransport = newRLPX
	}
	if srv.Dialer == nil {
		srv.Dialer = TCPDialer{&net.Dialer{Timeout: DefaultDialTimeout}}
	}
	srv.Quit = make(chan struct{})
	srv.Addpeer = make(chan *Conn)
	srv.Delpeer = make(chan PeerDrop)
	srv.Posthandshake = make(chan *Conn)
	srv.Addstatic = make(chan *discover.Node)
	srv.Removestatic = make(chan *discover.Node)
	srv.Addtrusted = make(chan *discover.Node)
	srv.Removetrusted = make(chan *discover.Node)
	srv.PeerOp = make(chan PeerOpFunc)
	srv.PeerOpDone = make(chan struct{})

	var (
		conn      *net.UDPConn
		sconn     *SharedUDPConn
		realaddr  *net.UDPAddr
		unhandled chan discover.ReadPacket
	)

	if !srv.NoDiscovery || srv.DiscoveryV5 {
		addr, err := net.ResolveUDPAddr("udp", srv.ListenAddr)
		if err != nil {
			return err
		}
		conn, err = net.ListenUDP("udp", addr)
		if err != nil {
			return err
		}
		realaddr = conn.LocalAddr().(*net.UDPAddr)
		if srv.NAT != nil {
			if !realaddr.IP.IsLoopback() {
				go nat.Map(srv.NAT, srv.Quit, "udp", realaddr.Port, realaddr.Port, "ethereum discovery")
			}
			// TODO: react to external IP changes over time.
			if ext, err := srv.NAT.ExternalIP(); err == nil {
				realaddr = &net.UDPAddr{IP: ext, Port: realaddr.Port}
			}
		}
	}

	if !srv.NoDiscovery && srv.DiscoveryV5 {
		unhandled = make(chan discover.ReadPacket, 100)
		sconn = &SharedUDPConn{conn, unhandled}
	}

	// node table
	if !srv.NoDiscovery {
		cfg := discover.Config{
			PrivateKey:   srv.PrivateKey,
			AnnounceAddr: realaddr,
			NodeDBPath:   srv.NodeDatabase,
			NetRestrict:  srv.NetRestrict,
			Bootnodes:    srv.BootstrapNodes,
			Unhandled:    unhandled,
		}
		ntab, err := discover.ListenUDP(conn, cfg)
		if err != nil {
			return err
		}
		srv.Ntab = ntab
	}

	if srv.DiscoveryV5 {
		var (
			ntab *discv5.Network
			err  error
		)
		if sconn != nil {
			ntab, err = discv5.ListenUDP(srv.PrivateKey, sconn, realaddr, "", srv.NetRestrict) //srv.NodeDatabase)
		} else {
			ntab, err = discv5.ListenUDP(srv.PrivateKey, conn, realaddr, "", srv.NetRestrict) //srv.NodeDatabase)
		}
		if err != nil {
			return err
		}
		if err := ntab.SetFallbackNodes(srv.BootstrapNodesV5); err != nil {
			return err
		}
		srv.DiscV5 = ntab
	}

	dynPeers := srv.maxDialedConns()
	dialer := newDialState(srv.StaticNodes, srv.BootstrapNodes, srv.Ntab, dynPeers, srv.NetRestrict)

	// handshake
	srv.OurHandshake = &ProtoHandshake{Version: BaseProtocolVersion, Name: srv.Name, ID: discover.PubkeyID(&srv.PrivateKey.PublicKey)}
	for _, p := range srv.Protocols {
		srv.OurHandshake.Caps = append(srv.OurHandshake.Caps, p.Cap())
	}
	// listen/dial
	if srv.ListenAddr != "" {
		if err := srv.startListening(); err != nil {
			return err
		}
	}
	if srv.NoDial && srv.ListenAddr == "" {
		srv.Log.Warn("P2P server will be useless, neither dialing nor listening")
	}

	srv.LoopWG.Add(1)
	go srv.run(dialer)
	srv.Running = true
	return nil
}

func (srv *Server) startListening() error {
	// Launch the TCP listener.
	listener, err := net.Listen("tcp", srv.ListenAddr)
	if err != nil {
		return err
	}
	laddr := listener.Addr().(*net.TCPAddr)
	srv.ListenAddr = laddr.String()
	srv.Listener = listener
	srv.LoopWG.Add(1)
	go srv.listenLoop()
	// Map the TCP listening port if NAT is configured.
	if !laddr.IP.IsLoopback() && srv.NAT != nil {
		srv.LoopWG.Add(1)
		go func() {
			nat.Map(srv.NAT, srv.Quit, "tcp", laddr.Port, laddr.Port, "ethereum p2p")
			srv.LoopWG.Done()
		}()
	}
	return nil
}

type Dialer interface {
	NewTasks(running int, peers map[discover.NodeID]*Peer, now time.Time) []task
	TaskDone(task, time.Time)
	AddStatic(*discover.Node)
	RemoveStatic(*discover.Node)
}

func (srv *Server) run(dialstate Dialer) {
	defer srv.LoopWG.Done()
	var (
		peers        = make(map[discover.NodeID]*Peer)
		inboundCount = 0
		trusted      = make(map[discover.NodeID]bool, len(srv.TrustedNodes))
		taskdone     = make(chan task, MaxActiveDialTasks)
		runningTasks []task
		queuedTasks  []task // tasks that can't run yet
	)
	// Put trusted nodes into a map to speed up checks.
	// Trusted peers are loaded on startup or added via AddTrustedPeer RPC.
	for _, n := range srv.TrustedNodes {
		trusted[n.ID] = true
	}

	// removes t from runningTasks
	delTask := func(t task) {
		for i := range runningTasks {
			if runningTasks[i] == t {
				runningTasks = append(runningTasks[:i], runningTasks[i+1:]...)
				break
			}
		}
	}
	// starts until max number of active tasks is satisfied
	startTasks := func(ts []task) (rest []task) {
		i := 0
		for ; len(runningTasks) < MaxActiveDialTasks && i < len(ts); i++ {
			t := ts[i]
			srv.Log.Trace("New dial task", "task", t)
			go func() { t.Do(srv); taskdone <- t }()
			runningTasks = append(runningTasks, t)
		}
		return ts[i:]
	}
	scheduleTasks := func() {
		// Start from queue first.
		queuedTasks = append(queuedTasks[:0], startTasks(queuedTasks)...)
		// Query dialer for new tasks and start as many as possible now.
		if len(runningTasks) < MaxActiveDialTasks {
			nt := dialstate.NewTasks(len(runningTasks)+len(queuedTasks), peers, time.Now())
			queuedTasks = append(queuedTasks, startTasks(nt)...)
		}
	}

running:
	for {
		scheduleTasks()

		select {
		case <-srv.Quit:
			// The server was stopped. Run the cleanup logic.
			break running
		case n := <-srv.Addstatic:
			// This channel is used by AddPeer to add to the
			// ephemeral static peer list. Add it to the dialer,
			// it will keep the node connected.
			srv.Log.Debug("Adding static node", "node", n)
			dialstate.AddStatic(n)
		case n := <-srv.Removestatic:
			// This channel is used by RemovePeer to send a
			// disconnect request to a peer and begin the
			// stop keeping the node connected.
			srv.Log.Debug("Removing static node", "node", n)
			dialstate.RemoveStatic(n)
			if p, ok := peers[n.ID]; ok {
				p.Disconnect(DiscRequested)
			}
		case n := <-srv.Addtrusted:
			// This channel is used by AddTrustedPeer to add an enode
			// to the trusted node set.
			srv.Log.Trace("Adding trusted node", "node", n)
			trusted[n.ID] = true
			// Mark any already-connected peer as trusted
			if p, ok := peers[n.ID]; ok {
				p.rw.Set(TrustedConn, true)
			}
		case n := <-srv.Removetrusted:
			// This channel is used by RemoveTrustedPeer to remove an enode
			// from the trusted node set.
			srv.Log.Trace("Removing trusted node", "node", n)
			delete(trusted, n.ID)
			// Unmark any already-connected peer as trusted
			if p, ok := peers[n.ID]; ok {
				p.rw.Set(TrustedConn, false)
			}
		case op := <-srv.PeerOp:
			// This channel is used by Peers and PeerCount.
			op(peers)
			srv.PeerOpDone <- struct{}{}
		case t := <-taskdone:
			// A task got done. Tell dialstate about it so it
			// can update its state and remove it from the active
			// tasks list.
			srv.Log.Trace("Dial task done", "task", t)
			dialstate.TaskDone(t, time.Now())
			delTask(t)
		case c := <-srv.Posthandshake:
			// A connection has passed the encryption handshake so
			// the remote identity is known (but hasn't been verified yet).
			if trusted[c.Id] {
				// Ensure that the trusted flag is set before checking against MaxPeers.
				c.Flags |= TrustedConn
			}
			// TODO: track in-progress inbound node IDs (pre-Peer) to avoid dialing them.
			select {
			case c.Cont <- srv.encHandshakeChecks(peers, inboundCount, c):
			case <-srv.Quit:
				break running
			}
		case c := <-srv.Addpeer:
			// At this point the connection is past the protocol handshake.
			// Its capabilities are known and the remote identity is verified.
			err := srv.protoHandshakeChecks(peers, inboundCount, c)
			if err == nil {
				// The handshakes are done and it passed all checks.
				p := newPeer(c, srv.Protocols)
				// If message events are enabled, pass the peerFeed
				// to the peer
				if srv.EnableMsgEvents {
					p.events = &srv.PeerFeed
				}
				name := truncateName(c.Name)

				go srv.runPeer(p)
				if peers[c.Id] != nil {
					peers[c.Id].PairPeer = p
					srv.Log.Debug("Adding p2p pair peer", "name", name, "addr", c.Fd.RemoteAddr(), "peers", len(peers)+1)
				} else {
					peers[c.Id] = p
					srv.Log.Debug("Adding p2p peer", "name", name, "addr", c.Fd.RemoteAddr(), "peers", len(peers)+1)
				}
				if p.Inbound() {
					inboundCount++
					serveSuccessMeter.Mark(1)
				} else {
					dialSuccessMeter.Mark(1)
				}
				activePeerGauge.Inc(1)
			}
			// The dialer logic relies on the assumption that
			// dial tasks complete after the peer has been added or
			// discarded. Unblock the task last.
			select {
			case c.Cont <- err:
			case <-srv.Quit:
				break running
			}
		case pd := <-srv.Delpeer:
			// A peer disconnected.
			d := common.PrettyDuration(mclock.Now() - pd.created)
			pd.log.Debug("Removing p2p peer", "duration", d, "peers", len(peers)-1, "req", pd.Requested, "err", pd.Err)
			delete(peers, pd.ID())
			if pd.Inbound() {
				inboundCount--
			}
			activePeerGauge.Dec(1)
		}
	}

	srv.Log.Trace("P2P networking is spinning down")

	// Terminate discovery. If there is a running lookup it will terminate soon.
	if srv.Ntab != nil {
		srv.Ntab.Close()
	}
	if srv.DiscV5 != nil {
		srv.DiscV5.Close()
	}
	// Disconnect all peers.
	for _, p := range peers {
		p.Disconnect(DiscQuitting)
	}
	// Wait for peers to shut down. Pending connections and tasks are
	// not handled here and will terminate soon-ish because srv.Quit
	// is closed.
	for len(peers) > 0 {
		p := <-srv.Delpeer
		p.GetLog().Trace("<-delpeer (spindown)", "remainingTasks", len(runningTasks))
		delete(peers, p.ID())
	}
}

func (srv *Server) protoHandshakeChecks(peers map[discover.NodeID]*Peer, inboundCount int, c *Conn) error {
	// Drop connections with no matching protocols.
	if len(srv.Protocols) > 0 && countMatchingProtocols(srv.Protocols, c.Caps) == 0 {
		return DiscUselessPeer
	}
	// Repeat the encryption handshake checks because the
	// peer set might have changed between the handshakes.
	return srv.encHandshakeChecks(peers, inboundCount, c)
}

func (srv *Server) encHandshakeChecks(peers map[discover.NodeID]*Peer, inboundCount int, c *Conn) error {
	switch {
	case !c.Is(TrustedConn|StaticDialedConn) && len(peers) >= srv.MaxPeers:
		return DiscTooManyPeers
	case !c.Is(TrustedConn) && c.Is(InboundConn) && inboundCount >= srv.maxInboundConns():
		return DiscTooManyPeers
	case peers[c.Id] != nil:
		exitPeer := peers[c.Id]
		if exitPeer.PairPeer != nil {
			return DiscAlreadyConnected
		}
		return nil
	case c.Id == srv.Self().ID:
		return DiscSelf
	default:
		return nil
	}
}

func (srv *Server) maxInboundConns() int {
	return srv.MaxPeers - srv.maxDialedConns()
}

func (srv *Server) maxDialedConns() int {
	if srv.NoDiscovery || srv.NoDial {
		return 0
	}
	r := srv.DialRatio
	if r == 0 {
		r = DefaultDialRatio
	}
	return srv.MaxPeers / r
}

type TempError interface {
	Temporary() bool
}

// listenLoop runs in its own goroutine and accepts
// inbound connections.
func (srv *Server) listenLoop() {
	defer srv.LoopWG.Done()
	srv.Log.Info("RLPx listener up", "self", srv.makeSelf(srv.Listener, srv.Ntab))

	tokens := DefaultMaxPendingPeers
	if srv.MaxPendingPeers > 0 {
		tokens = srv.MaxPendingPeers
	}
	slots := make(chan struct{}, tokens)
	for i := 0; i < tokens; i++ {
		slots <- struct{}{}
	}

	for {
		// Wait for a handshake slot before accepting.
		<-slots

		var (
			fd  net.Conn
			err error
		)
		for {
			fd, err = srv.Listener.Accept()
			if tempErr, ok := err.(TempError); ok && tempErr.Temporary() {
				srv.Log.Debug("Temporary read error", "err", err)
				continue
			} else if err != nil {
				srv.Log.Debug("Read error", "err", err)
				return
			}
			break
		}

		// Reject connections that do not match NetRestrict.
		if srv.NetRestrict != nil {
			if tcp, ok := fd.RemoteAddr().(*net.TCPAddr); ok && !srv.NetRestrict.Contains(tcp.IP) {
				srv.Log.Debug("Rejected conn (not whitelisted in NetRestrict)", "addr", fd.RemoteAddr())
				fd.Close()
				slots <- struct{}{}
				continue
			}
		}

		fd = newMeteredConn(fd)
		serveMeter.Mark(1)
		srv.Log.Trace("Accepted connection", "addr", fd.RemoteAddr())
		go func() {
			srv.SetupConn(fd, InboundConn, nil)
			slots <- struct{}{}
		}()
	}
}

// SetupConn runs the handshakes and attempts to add the connection
// as a peer. It returns when the connection has been added as a peer
// or the handshakes have failed.
func (srv *Server) SetupConn(fd net.Conn, flags ConnFlag, dialDest *discover.Node) error {
	self := srv.Self()
	if self == nil {
		return errors.New("shutdown")
	}
	c := &Conn{Fd: fd, Transport: srv.NewTransport(fd), Flags: flags, Cont: make(chan error)}
	err := srv.setupConn(c, flags, dialDest)
	if err != nil {
		if !c.Is(InboundConn) {
			markDialError(err)
		}
		c.Close(err)
		srv.Log.Trace("Setting up connection failed", "id", c.Id, "err", err)
	}
	return err
}

func (srv *Server) setupConn(c *Conn, flags ConnFlag, dialDest *discover.Node) error {
	// Prevent leftover pending conns from entering the handshake.
	srv.Lock.Lock()
	running := srv.Running
	srv.Lock.Unlock()
	if !running {
		return ErrServerStopped
	}
	// Run the encryption handshake.
	var err error
	if c.Id, err = c.DoEncHandshake(srv.PrivateKey, dialDest); err != nil {
		srv.Log.Trace("Failed RLPx handshake", "addr", c.Fd.RemoteAddr(), "conn", c.Flags, "err", err)
		return err
	}
	clog := srv.Log.New("id", c.Id, "addr", c.Fd.RemoteAddr(), "conn", c.Flags)
	// For dialed connections, check that the remote public key matches.
	if dialDest != nil && c.Id != dialDest.ID {
		clog.Trace("Dialed identity mismatch", "want", c, dialDest.ID)
		return DiscUnexpectedIdentity
	}
	err = srv.checkpoint(c, srv.Posthandshake)
	if err != nil {
		clog.Trace("Rejected peer before protocol handshake", "err", err)
		return err
	}
	// Run the protocol handshake
	phs, err := c.DoProtoHandshake(srv.OurHandshake)
	if err != nil {
		clog.Trace("Failed proto handshake", "err", err)
		return err
	}
	if phs.ID != c.Id {
		clog.Trace("Wrong devp2p handshake identity", "err", phs.ID)
		return DiscUnexpectedIdentity
	}
	c.Caps, c.Name = phs.Caps, phs.Name
	err = srv.checkpoint(c, srv.Addpeer)
	if err != nil {
		clog.Trace("Rejected peer", "err", err)
		return err
	}
	// If the checks completed successfully, runPeer has now been
	// launched by run.
	clog.Trace("connection set up", "inbound", dialDest == nil)
	return nil
}

func truncateName(s string) string {
	if len(s) > 20 {
		return s[:20] + "..."
	}
	return s
}

// checkpoint sends the conn to run, which performs the
// post-handshake checks for the stage (posthandshake, addpeer).
func (srv *Server) checkpoint(c *Conn, stage chan<- *Conn) error {
	select {
	case stage <- c:
	case <-srv.Quit:
		return ErrServerStopped
	}
	select {
	case err := <-c.Cont:
		return err
	case <-srv.Quit:
		return ErrServerStopped
	}
}

// runPeer runs in its own goroutine for each peer.
// it waits until the Peer logic returns and removes
// the peer.
func (srv *Server) runPeer(p *Peer) {
	if srv.NewPeerHook != nil {
		srv.NewPeerHook(p)
	}

	// broadcast peer add
	srv.PeerFeed.Send(&PeerEvent{
		Type: PeerEventTypeAdd,
		Peer: p.ID(),
	})

	// run the protocol
	remoteRequested, err := p.run()

	// broadcast peer drop
	srv.PeerFeed.Send(&PeerEvent{
		Type:  PeerEventTypeDrop,
		Peer:  p.ID(),
		Error: err.Error(),
	})

	// Note: run waits for existing peers to be sent on srv.delpeer
	// before returning, so this send should not select on srv.quit.
	srv.Delpeer <- PeerDrop{p, err, remoteRequested}
}

// NodeInfo represents a short summary of the information known about the host.
type NodeInfo struct {
	ID    string `json:"id"`    // Unique node identifier (also the encryption key)
	Name  string `json:"name"`  // Name of the node, including client type, version, OS, custom data
	Enode string `json:"enode"` // Enode URL for adding this peer from remote peers
	IP    string `json:"ip"`    // IP address of the node
	Ports struct {
		Discovery int `json:"discovery"` // UDP listening port for discovery protocol
		Listener  int `json:"listener"`  // TCP listening port for RLPx
	} `json:"ports"`
	ListenAddr string                 `json:"listenAddr"`
	Protocols  map[string]interface{} `json:"protocols"`
}

// NodeInfo gathers and returns a collection of metadata known about the host.
func (srv *Server) NodeInfo() *NodeInfo {
	node := srv.Self()

	// Gather and assemble the generic node infos
	info := &NodeInfo{
		Name:       srv.Name,
		Enode:      node.String(),
		ID:         node.ID.String(),
		IP:         node.IP.String(),
		ListenAddr: srv.ListenAddr,
		Protocols:  make(map[string]interface{}),
	}
	info.Ports.Discovery = int(node.UDP)
	info.Ports.Listener = int(node.TCP)

	// Gather all the running protocol infos (only once per protocol type)
	for _, proto := range srv.Protocols {
		if _, ok := info.Protocols[proto.Name]; !ok {
			nodeInfo := interface{}("unknown")
			if query := proto.NodeInfo; query != nil {
				nodeInfo = proto.NodeInfo()
			}
			info.Protocols[proto.Name] = nodeInfo
		}
	}
	return info
}

// PeersInfo returns an array of metadata objects describing connected peers.
func (srv *Server) PeersInfo() []*PeerInfo {
	// Gather all the generic and sub-protocol specific infos
	infos := make([]*PeerInfo, 0, srv.PeerCount())
	for _, peer := range srv.Peers() {
		if peer != nil {
			infos = append(infos, peer.Info())
		}
	}
	// Sort the result array alphabetically by node identifier
	for i := 0; i < len(infos); i++ {
		for j := i + 1; j < len(infos); j++ {
			if infos[i].ID > infos[j].ID {
				infos[i], infos[j] = infos[j], infos[i]
			}
		}
	}
	return infos
}
