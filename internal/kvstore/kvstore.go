// Package kvstore implements a distributed key-value store using consistent hashing.
//
// Keys are mapped to nodes via consistent hashing, then replicated to a
// configurable number of replicas. Each store node runs as a goroutine
// communicating via channels over an in-memory transport.
package kvstore

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/atlas-engine/atlas/internal/algorithms/hashing"
)

// Operation represents a KV store operation.
type Operation string

const (
	OpPut    Operation = "PUT"
	OpGet    Operation = "GET"
	OpDelete Operation = "DELETE"
)

// Request is a message sent to a store node.
type Request struct {
	Op    Operation
	Key   string
	Value string
	Resp  chan Response
}

// Response is the result of a KV store operation.
type Response struct {
	Value string
	Found bool
	Err   error
}

// Node represents a single KV store node.
type Node struct {
	ID     string
	Addr   string
	data   map[string]string
	inbox  chan Request
	mu     sync.RWMutex
}

// Store is a distributed KV store.
type Store struct {
	ring         *hashing.HashRing
	nodes        map[string]*Node
	replicationFactor int
	log          *slog.Logger
	requestCounter atomic.Int64
}

// New creates a new distributed KV store.
func New(replicationFactor int, log *slog.Logger) *Store {
	if replicationFactor < 1 {
		replicationFactor = 1
	}
	return &Store{
		ring:              hashing.NewHashRing(150),
		nodes:             make(map[string]*Node),
		replicationFactor: replicationFactor,
		log:               log,
	}
}

// AddNode adds a store node. The node starts serving requests immediately.
func (s *Store) AddNode(id, addr string) {
	node := &Node{
		ID:    id,
		Addr:  addr,
		data:  make(map[string]string),
		inbox: make(chan Request, 1000),
	}
	s.nodes[id] = node
	s.ring.AddNode(id)
	go s.nodeLoop(node)

	if s.log != nil {
		s.log.Info("kv store node added", slog.String("node", id), slog.String("addr", addr))
	}
}

// RemoveNode removes a store node and stops serving.
func (s *Store) RemoveNode(id string) {
	node, ok := s.nodes[id]
	if !ok {
		return
	}
	s.ring.RemoveNode(id)
	delete(s.nodes, id)
	close(node.inbox)
	if s.log != nil {
		s.log.Info("kv store node removed", slog.String("node", id))
	}
}

// Put stores a key-value pair with replication.
func (s *Store) Put(ctx context.Context, key, value string) error {
	replicas := s.ring.GetNodes(key, s.replicationFactor)
	if len(replicas) == 0 {
		return fmt.Errorf("no nodes available")
	}

	for _, replicaID := range replicas {
		node, ok := s.nodes[replicaID]
		if !ok {
			continue
		}
		resp := s.call(ctx, node, Operation(OpPut), key, value)
		if resp.Err != nil {
			return resp.Err
		}
	}
	return nil
}

// Get retrieves a value, checking replicas for availability.
func (s *Store) Get(ctx context.Context, key string) (string, bool, error) {
	replicas := s.ring.GetNodes(key, s.replicationFactor)
	if len(replicas) == 0 {
		return "", false, fmt.Errorf("no nodes available")
	}

	var lastErr error
	for _, replicaID := range replicas {
		node, ok := s.nodes[replicaID]
		if !ok {
			lastErr = fmt.Errorf("replica %s not found", replicaID)
			continue
		}
		resp := s.call(ctx, node, Operation(OpGet), key, "")
		if resp.Err != nil {
			lastErr = resp.Err
			continue
		}
		if resp.Found {
			return resp.Value, true, nil
		}
	}
	return "", false, lastErr
}

// Delete removes a key across all replicas.
func (s *Store) Delete(ctx context.Context, key string) error {
	replicas := s.ring.GetNodes(key, s.replicationFactor)
	if len(replicas) == 0 {
		return fmt.Errorf("no nodes available")
	}

	for _, replicaID := range replicas {
		node, ok := s.nodes[replicaID]
		if !ok {
			continue
		}
		resp := s.call(ctx, node, Operation(OpDelete), key, "")
		if resp.Err != nil {
			return resp.Err
		}
	}
	return nil
}

// Keys returns all keys stored across all nodes.
func (s *Store) Keys() []string {
	keySet := make(map[string]bool)
	for _, node := range s.nodes {
		node.mu.RLock()
		for k := range node.data {
			keySet[k] = true
		}
		node.mu.RUnlock()
	}
	keys := make([]string, 0, len(keySet))
	for k := range keySet {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// NodeCount returns the number of active nodes.
func (s *Store) NodeCount() int {
	return len(s.nodes)
}

// NodeIDs returns sorted node IDs for visualization.
func (s *Store) NodeIDs() []string {
	ids := make([]string, 0, len(s.nodes))
	for id := range s.nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// NodeForKey returns which node owns a key.
func (s *Store) NodeForKey(key string) string {
	return s.ring.GetNode(key)
}

func (s *Store) call(ctx context.Context, node *Node, op Operation, key, value string) Response {
	req := Request{
		Op:    op,
		Key:   key,
		Value: value,
		Resp:  make(chan Response, 1),
	}

	select {
	case node.inbox <- req:
	case <-ctx.Done():
		return Response{Err: ctx.Err()}
	}

	select {
	case resp := <-req.Resp:
		return resp
	case <-ctx.Done():
		return Response{Err: ctx.Err()}
	}
}

func (s *Store) nodeLoop(node *Node) {
	for req := range node.inbox {
		s.processRequest(node, req)
	}
}

func (s *Store) processRequest(node *Node, req Request) {
	switch req.Op {
	case OpPut:
		node.mu.Lock()
		node.data[req.Key] = req.Value
		node.mu.Unlock()
		req.Resp <- Response{Found: true}
	case OpGet:
		node.mu.RLock()
		value, ok := node.data[req.Key]
		node.mu.RUnlock()
		req.Resp <- Response{Value: value, Found: ok}
	case OpDelete:
		node.mu.Lock()
		delete(node.data, req.Key)
		node.mu.Unlock()
		req.Resp <- Response{Found: true}
	}
}