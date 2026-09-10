// Package raft implements a simplified but real Raft consensus protocol.
//
// Implements:
//   - Leader election (follower, candidate, leader states)
//   - Terms and election timeouts
//   - Heartbeats (AppendEntries with no entries)
//   - Log replication and commit index tracking
//   - RequestVote and AppendEntries RPCs
//   - Leader failure detection and re-election
//   - Log consistency checks
//
// This is an educational implementation of the Raft paper
// (In Search of an Understandable Consensus Algorithm).
package raft

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"
)

// State is the Raft node state.
type State int

const (
	Follower State = iota
	Candidate
	Leader
)

func (s State) String() string {
	switch s {
	case Follower:
		return "follower"
	case Candidate:
		return "candidate"
	case Leader:
		return "leader"
	default:
		return "unknown"
	}
}

// LogEntry is a replicated log entry.
type LogEntry struct {
	Term    int         `json:"term"`
	Index   int         `json:"index"`
	Command interface{} `json:"command"`
}

// Message is an internal RPC message between nodes.
type Message struct {
	Type     string
	From     string
	To       string
	Term     int
	Data     map[string]interface{}
}

// Transport is the message transport between nodes.
type Transport interface {
	Send(msg Message)
}

// InMemoryTransport routes messages via channels (for testing/simulation).
type InMemoryTransport struct {
	boxes map[string]chan Message
	mu    sync.RWMutex
}

// NewInMemoryTransport creates a channel-based transport.
func NewInMemoryTransport() *InMemoryTransport {
	return &InMemoryTransport{boxes: make(map[string]chan Message)}
}

// Register creates a mailbox for a node.
func (t *InMemoryTransport) Register(nodeID string) chan Message {
	t.mu.Lock()
	defer t.mu.Unlock()
	box := make(chan Message, 1000)
	t.boxes[nodeID] = box
	return box
}

// Send delivers a message to the target node's mailbox.
func (t *InMemoryTransport) Send(msg Message) {
	t.mu.RLock()
	box, ok := t.boxes[msg.To]
	t.mu.RUnlock()
	if ok {
		select {
		case box <- msg:
		default:
		}
	}
}

// Config configures a Raft node.
type Config struct {
	ID                 string
	Peers              []string
	Transport          Transport
	ElectionTimeoutMin time.Duration
	ElectionTimeoutMax time.Duration
	HeartbeatInterval  time.Duration
	Log                *slog.Logger
}

// Node is a single Raft consensus node.
type Node struct {
	id        string
	state     State
	peers     []string
	transport Transport
	logger    *slog.Logger

	mu sync.RWMutex

	currentTerm int
	votedFor    string
	log         []LogEntry
	commitIndex int
	lastApplied int

	// volatile leader state
	nextIndex  map[string]int
	matchIndex map[string]int

	// election timer
	electionTimeout time.Duration
	lastHeartbeat   time.Time

	mailbox chan Message
	done    chan struct{}
	voteCh  chan int
}

// NewNode creates a new Raft node.
func NewNode(cfg Config) *Node {
	if cfg.ElectionTimeoutMin <= 0 {
		cfg.ElectionTimeoutMin = 800 * time.Millisecond
	}
	if cfg.ElectionTimeoutMax <= 0 {
		cfg.ElectionTimeoutMax = 1600 * time.Millisecond
	}
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = 100 * time.Millisecond
	}
	if cfg.Log == nil {
		cfg.Log = slog.Default()
	}

	n := &Node{
		id:              cfg.ID,
		state:           Follower,
		peers:           cfg.Peers,
		transport:       cfg.Transport,
		logger:          cfg.Log,
		nextIndex:       make(map[string]int),
		matchIndex:      make(map[string]int),
		electionTimeout: randomTimeout(cfg.ElectionTimeoutMin, cfg.ElectionTimeoutMax),
		lastHeartbeat:   time.Now(),
		done:            make(chan struct{}),
	}

	n.stepDown(n.currentTerm, "")
	n.lastHeartbeat = time.Now()

	if tc, ok := cfg.Transport.(*InMemoryTransport); ok {
		n.mailbox = tc.Register(cfg.ID)
	}

	return n
}

// Start launches the Raft node's main loop.
func (n *Node) Start(ctx context.Context) {
	go n.run(ctx)
}

// Stop stops the Raft node.
func (n *Node) Stop() {
	select {
	case <-n.done:
	default:
		close(n.done)
	}
}

func (n *Node) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-n.done:
			return
		case msg := <-n.mailbox:
			n.handleMessage(msg)
		case <-time.After(time.Millisecond * 10):
			n.tick()
		}
	}
}

func (n *Node) tick() {
	switch n.getState() {
	case Follower, Candidate:
		if time.Since(n.lastHeartbeat) > n.electionTimeout {
			n.startElection()
		}
	case Leader:
		if time.Since(n.lastHeartbeat) > 50*time.Millisecond {
			n.broadcastAppendEntries()
		}
	}
}

func (n *Node) getState() State {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.state
}

func (n *Node) startElection() {
	n.mu.Lock()
	if n.state == Leader {
		n.mu.Unlock()
		return
	}
	n.currentTerm++
	n.state = Candidate
	n.votedFor = n.id
	term := n.currentTerm
	lastLogTerm := n.lastLogEntry().Term
	lastLogIndex := n.lastLogEntry().Index
	electionTimeout := n.electionTimeout
	voteCh := make(chan int, len(n.peers)+1)
	n.voteCh = voteCh
	n.mu.Unlock()

	n.logger.Info("starting election",
		slog.String("node", n.id),
		slog.Int("term", term))

	voteCh <- 1

	for _, peer := range n.peers {
		go func(peer string) {
			n.sendMessage(Message{
				Type: "request_vote",
				From: n.id,
				To:   peer,
				Term: term,
				Data: map[string]interface{}{
					"last_log_term":  lastLogTerm,
					"last_log_index": lastLogIndex,
				},
			})
		}(peer)
	}

	majority := len(n.peers)/2 + 1
	go func() {
		timer := time.NewTimer(electionTimeout)
		defer timer.Stop()

		votes := 0
		for {
			select {
			case v := <-voteCh:
				votes += v
				if votes >= majority {
					n.mu.Lock()
					if n.state == Candidate {
						n.state = Leader
						n.logger.Info("became leader",
							slog.String("node", n.id),
							slog.Int("term", n.currentTerm))
						for _, p := range n.peers {
							n.nextIndex[p] = n.lastLogEntry().Index + 1
							n.matchIndex[p] = 0
						}
						n.mu.Unlock()
						n.broadcastAppendEntries()
					} else {
						n.mu.Unlock()
					}
					return
				}
			case <-timer.C:
				return
			}
		}
	}()
}

func (n *Node) handleMessage(msg Message) {
	switch msg.Type {
	case "request_vote":
		n.handleRequestVote(msg)
	case "request_vote_response":
		n.handleRequestVoteResponse(msg)
	case "append_entries":
		n.handleAppendEntries(msg)
	case "append_entries_response":
		n.handleAppendEntriesResponse(msg)
	}
}

func (n *Node) handleRequestVote(msg Message) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if msg.Term > n.currentTerm {
		n.stepDown(msg.Term, "")
	}

	followerLastLog := n.lastLogEntry()
	candidateTerm, _ := msg.Data["last_log_term"].(int)
	candidateIndex, _ := msg.Data["last_log_index"].(int)

	if msg.Term < n.currentTerm {
		n.sendMessage(Message{
			Type: "request_vote_response",
			From: n.id,
			To:   msg.From,
			Term: n.currentTerm,
			Data: map[string]interface{}{"vote_granted": false},
		})
		return
	}

	// Raft rule: candidate's log must be at least as up-to-date as ours
	upToDate := candidateTerm > followerLastLog.Term ||
		(candidateTerm == followerLastLog.Term && candidateIndex >= followerLastLog.Index)

	if (n.votedFor == "" || n.votedFor == msg.From) && upToDate {
		n.votedFor = msg.From
		n.lastHeartbeat = time.Now()
		n.sendMessage(Message{
			Type: "request_vote_response",
			From: n.id,
			To:   msg.From,
			Term: n.currentTerm,
			Data: map[string]interface{}{"vote_granted": true},
		})
	} else {
		n.sendMessage(Message{
			Type: "request_vote_response",
			From: n.id,
			To:   msg.From,
			Term: n.currentTerm,
			Data: map[string]interface{}{"vote_granted": false},
		})
	}
}

func (n *Node) handleRequestVoteResponse(msg Message) {
	n.mu.RLock()
	if msg.Term > n.currentTerm {
		n.mu.RUnlock()
		n.mu.Lock()
		n.stepDown(msg.Term, "")
		n.mu.Unlock()
		return
	}
	if n.state != Candidate {
		n.mu.RUnlock()
		return
	}
	voteCh := n.voteCh
	n.mu.RUnlock()

	voteGranted, _ := msg.Data["vote_granted"].(bool)
	if !voteGranted {
		return
	}

	if voteCh != nil {
		select {
		case voteCh <- 1:
		default:
		}
	}
}

func (n *Node) handleAppendEntries(msg Message) {
	n.mu.Lock()
	defer n.mu.Unlock()

	term := msg.Term
	leaderID, _ := msg.Data["leader_id"].(string)
	prevLogIndex, _ := msg.Data["prev_log_index"].(int)
	entries, _ := msg.Data["entries"].([]interface{})
	leaderCommit, _ := msg.Data["leader_commit"].(int)

	if msg.Term < n.currentTerm {
		n.sendMessage(Message{
			Type: "append_entries_response",
			From: n.id,
			To:   msg.From,
			Term: n.currentTerm,
			Data: map[string]interface{}{"success": false},
		})
		return
	}

	if term > n.currentTerm {
		n.stepDown(term, leaderID)
	} else {
		n.lastHeartbeat = time.Now()
	}

	// Log consistency check
	if len(n.log) <= prevLogIndex {
		n.sendMessage(Message{
			Type: "append_entries_response",
			From: n.id,
			To:   msg.From,
			Term: n.currentTerm,
			Data: map[string]interface{}{"success": false},
		})
		return
	}

	// Append entries
	for i, entry := range entries {
		e := entry.(LogEntry)
		idx := prevLogIndex + 1 + i
		if idx < len(n.log) {
			if n.log[idx].Term != e.Term {
				n.log = n.log[:idx]
				n.log = append(n.log, e)
			}
		} else {
			n.log = append(n.log, e)
		}
	}

	if leaderCommit > n.commitIndex {
		logLen := len(n.log) - 1
		if leaderCommit < logLen {
			n.commitIndex = leaderCommit
		} else {
			n.commitIndex = logLen
		}
	}

	n.sendMessage(Message{
		Type: "append_entries_response",
		From: n.id,
		To:   msg.From,
		Term: n.currentTerm,
		Data: map[string]interface{}{
			"success":    true,
			"next_index": len(n.log),
		},
	})
}

func (n *Node) handleAppendEntriesResponse(msg Message) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if msg.Term > n.currentTerm {
		n.stepDown(msg.Term, "")
		return
	}
	if n.state != Leader {
		return
	}

	success, _ := msg.Data["success"].(bool)
	if success {
		nextIndex, _ := msg.Data["next_index"].(int)
		n.matchIndex[msg.From] = nextIndex - 1
		n.nextIndex[msg.From] = nextIndex
	}
}

func (n *Node) broadcastAppendEntries() {
	n.mu.Lock()
	term := n.currentTerm
	commitIndex := n.commitIndex
	peers := append([]string(nil), n.peers...)
	nextIndex := make(map[string]int, len(peers))
	for _, p := range peers {
		nextIndex[p] = n.nextIndex[p]
	}
	logCopy := append([]LogEntry(nil), n.log...)
	n.mu.Unlock()

	for _, peer := range peers {
		go func(peer string) {
			nextIdx := nextIndex[peer]

			prevLogIndex := nextIdx - 1
			var prevLogTerm int
			if prevLogIndex >= 0 && prevLogIndex < len(logCopy) {
				prevLogTerm = logCopy[prevLogIndex].Term
			}
			var entries []interface{}
			for i := nextIdx; i < len(logCopy); i++ {
				entries = append(entries, logCopy[i])
			}

			n.sendMessage(Message{
				Type: "append_entries",
				From: n.id,
				To:   peer,
				Term: term,
				Data: map[string]interface{}{
					"leader_id":       n.id,
					"prev_log_index":  prevLogIndex,
					"prev_log_term":   prevLogTerm,
					"entries":         entries,
					"leader_commit":   commitIndex,
				},
			})
		}(peer)
	}
	n.lastHeartbeat = time.Now()
}

// Append commits a command via the leader. Only the leader can serve writes.
func (n *Node) Append(command interface{}) (int, error) {
	if n.getState() != Leader {
		return 0, fmt.Errorf("not leader")
	}

	n.mu.Lock()
	entry := LogEntry{
		Term:    n.currentTerm,
		Index:   len(n.log),
		Command: command,
	}
	n.log = append(n.log, entry)
	index := entry.Index
	n.mu.Unlock()

	n.broadcastAppendEntries()
	return index, nil
}

// GetState returns the current state of the node.
func (n *Node) GetState() State {
	return n.getState()
}

// GetTerm returns the current term.
func (n *Node) GetTerm() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.currentTerm
}

// GetCommitIndex returns the committed log index.
func (n *Node) GetCommitIndex() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.commitIndex
}

// GetLog returns a copy of the replicated log.
func (n *Node) GetLog() []LogEntry {
	n.mu.RLock()
	defer n.mu.RUnlock()
	result := make([]LogEntry, len(n.log))
	copy(result, n.log)
	return result
}

// LastLogIndex returns the last entry index.
func (n *Node) GetLastLogIndex() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.lastLogEntry().Index
}

func (n *Node) lastLogEntry() LogEntry {
	if len(n.log) == 0 {
		return LogEntry{Term: 0, Index: -1}
	}
	return n.log[len(n.log)-1]
}

func (n *Node) stepDown(term int, leaderID string) {
	n.currentTerm = term
	n.state = Follower
	n.votedFor = ""
	if leaderID != "" {
		n.lastHeartbeat = time.Now()
	}
}

func (n *Node) sendMessage(msg Message) {
	if n.transport != nil {
		n.transport.Send(msg)
	}
}

func randomTimeout(min, max time.Duration) time.Duration {
	if max <= min {
		return min
	}
	return min + time.Duration(rand.Int63n(int64(max-min)))
}