package raft

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

type logger interface {
	Fatalf(format string, args ...any)
	Fatal(args ...any)
	Helper()
}

func defaultTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(new(discardWriter), nil))
}

type discardWriter struct{}

func (d *discardWriter) Write(p []byte) (int, error) { return len(p), nil }

func newRaftCluster(t logger, numNodes int) ([]*Node, *InMemoryTransport) {
	transport := NewInMemoryTransport()
	nodes := make([]*Node, 0, numNodes)
	peers := make([]string, 0, numNodes)

	for i := 0; i < numNodes; i++ {
		peers = append(peers, "node-"+string(rune('a'+i)))
	}

	for i := 0; i < numNodes; i++ {
		var otherPeers []string
		for j, p := range peers {
			if j != i {
				otherPeers = append(otherPeers, p)
			}
		}
		node := NewNode(Config{
			ID:                 peers[i],
			Peers:              otherPeers,
			Transport:          transport,
			ElectionTimeoutMin: 300 * time.Millisecond,
			ElectionTimeoutMax: 600 * time.Millisecond,
			HeartbeatInterval:  150 * time.Millisecond,
			Log:                defaultTestLogger(),
		})
		nodes = append(nodes, node)
	}

	return nodes, transport
}

func startAll(ctx context.Context, nodes []*Node) {
	for _, n := range nodes {
		n.Start(ctx)
	}
}

func waitForLeader(t logger, nodes []*Node, timeout time.Duration) *Node {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, n := range nodes {
			if n.GetState() == Leader {
				return n
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("no leader elected within timeout")
	return nil
}

func TestLeaderElection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes, _ := newRaftCluster(t, 3)
	startAll(ctx, nodes)

	leader := waitForLeader(t, nodes, 5*time.Second)
	if leader == nil {
		return
	}
	if leader.GetState() != Leader {
		t.Errorf("expected leader state, got %s", leader.GetState())
	}
	if leader.GetTerm() < 1 {
		t.Errorf("expected term >= 1, got %d", leader.GetTerm())
	}
}

func TestFollowersAndCandidate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes, _ := newRaftCluster(t, 3)
	startAll(ctx, nodes)

	waitForLeader(t, nodes, 5*time.Second)

	leaderCount := 0
	followerCount := 0
	for _, n := range nodes {
		switch n.GetState() {
		case Leader:
			leaderCount++
		case Follower:
			followerCount++
		}
	}

	if leaderCount != 1 {
		t.Errorf("expected exactly 1 leader, got %d", leaderCount)
	}
	if followerCount != len(nodes)-1 {
		t.Errorf("expected %d followers, got %d", len(nodes)-1, followerCount)
	}
}

func TestLeaderFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes, _ := newRaftCluster(t, 3)
	startAll(ctx, nodes)

	// Wait for initial leader
	initialLeader := waitForLeader(t, nodes, 5*time.Second)

	// Kill the leader
	initialLeader.Stop()
	for _, n := range nodes {
		if n != initialLeader {
			// Reset election deadlines so a new leader can be elected quickly
			n.lastHeartbeat = n.lastHeartbeat.Add(-time.Hour)
		}
	}

	// Wait for new leader
	deadline := time.Now().Add(5 * time.Second)
	newLeaderFound := false
	for time.Now().Before(deadline) {
		for _, n := range nodes {
			if n != initialLeader && n.GetState() == Leader {
				newLeaderFound = true
				break
			}
		}
		if newLeaderFound {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if !newLeaderFound {
		t.Fatal("system did not elect a new leader after leader failure")
	}
}

func TestLogReplication(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes, _ := newRaftCluster(t, 3)
	startAll(ctx, nodes)

	leader := waitForLeader(t, nodes, 5*time.Second)

	// Append entries via leader
	index, err := leader.Append("command-1")
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}
	leader.Append("command-2")

	// Give replication time
	time.Sleep(500 * time.Millisecond)

	// Check log replicas
	for _, n := range nodes {
		log := n.GetLog()
		if len(log) == 0 {
			continue
		}
		found := false
		for _, entry := range log {
			if entry.Index == index && entry.Command == "command-1" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("node %s missing replicated entry", n.id)
		}
	}
}

func TestSingleNode(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	transport := NewInMemoryTransport()
	node := NewNode(Config{
		ID:                 "node-a",
		Peers:              nil,
		Transport:          transport,
		ElectionTimeoutMin: 200 * time.Millisecond,
		ElectionTimeoutMax: 400 * time.Millisecond,
	})
	node.Start(ctx)

	time.Sleep(1 * time.Second)
	if node.GetState() != Leader {
		t.Errorf("single node should become leader, got %s", node.GetState())
	}
}

func TestAppendOnlyLeader(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes, _ := newRaftCluster(t, 3)
	startAll(ctx, nodes)

	leader := waitForLeader(t, nodes, 5*time.Second)

	follower := nodes[0]
	if follower == leader {
		follower = nodes[1]
	}

	_, err := follower.Append("should-fail")
	if err == nil {
		t.Error("expected error appending to follower")
	}

	_, err = leader.Append("should-succeed")
	if err != nil {
		t.Errorf("expected success appending to leader, got %v", err)
	}
}

func TestTermIncreases(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes, _ := newRaftCluster(t, 3)
	startAll(ctx, nodes)

	initialTerms := make([]int, len(nodes))
	for i, n := range nodes {
		initialTerms[i] = n.GetTerm()
	}

	time.Sleep(2 * time.Second)

	// Terms should have increased (at least one election happened)
	anyIncreased := false
	for i, n := range nodes {
		if n.GetTerm() > initialTerms[i] {
			anyIncreased = true
			break
		}
	}
	if !anyIncreased {
		t.Error("expected at least one node's term to increase")
	}
}

func TestElectionConvergence(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for round := 0; round < 5; round++ {
		nodes, transport := newRaftCluster(t, 3)
		startAll(ctx, nodes)

		leader := waitForLeader(t, nodes, 5*time.Second)
		_ = leader
		_ = transport
		time.Sleep(200 * time.Millisecond)
	}
}

func BenchmarkRaftAppend(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes, _ := newRaftCluster(b, 3)
	startAll(ctx, nodes)

	leader := waitForLeader(b, nodes, 5*time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		leader.Append(i)
	}
}