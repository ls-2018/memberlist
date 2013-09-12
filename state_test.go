package memberlist

import (
	"testing"
	"time"
)

func HostMemberlist(host string, t *testing.T) *Memberlist {
	c := DefaultConfig()
	c.Name = host
	c.BindAddr = host
	m, err := newMemberlist(c)
	if err != nil {
		t.Fatalf("failed to get memberlist: %s", err)
	}
	return m
}

func TestMemberList_Tick(t *testing.T) {
	m1 := HostMemberlist("127.0.0.50", t)
	m1.config.RTT = time.Millisecond
	m1.config.Interval = 10 * time.Millisecond
	m2 := HostMemberlist("127.0.0.51", t)

	a1 := alive{Node: "127.0.0.50", Addr: []byte{127, 0, 0, 50}, Incarnation: 1}
	m1.aliveNode(&a1)
	a2 := alive{Node: "127.0.0.51", Addr: []byte{127, 0, 0, 51}, Incarnation: 1}
	m1.aliveNode(&a2)

	// should ping 127.0.0.51
	m1.tick()

	// Should not be marked suspect
	n := m1.nodeMap["127.0.0.51"]
	if n.State != StateAlive {
		t.Fatalf("Expect node to be alive")
	}

	// Should increment seqno
	if m1.sequenceNum != 1 {
		t.Fatalf("bad seqno %v", m2.sequenceNum)
	}
}

func TestMemberList_ProbeNode_Suspect(t *testing.T) {
	m1 := HostMemberlist("127.0.0.100", t)
	m1.config.RTT = time.Millisecond
	m1.config.Interval = 10 * time.Millisecond
	m2 := HostMemberlist("127.0.0.101", t)
	m3 := HostMemberlist("127.0.0.102", t)

	a1 := alive{Node: "127.0.0.100", Addr: []byte{127, 0, 0, 100}, Incarnation: 1}
	m1.aliveNode(&a1)
	a2 := alive{Node: "127.0.0.101", Addr: []byte{127, 0, 0, 101}, Incarnation: 1}
	m1.aliveNode(&a2)
	a3 := alive{Node: "127.0.0.102", Addr: []byte{127, 0, 0, 102}, Incarnation: 1}
	m1.aliveNode(&a3)
	a4 := alive{Node: "127.0.0.103", Addr: []byte{127, 0, 0, 103}, Incarnation: 1}
	m1.aliveNode(&a4)

	n := m1.nodeMap["127.0.0.103"]
	m1.probeNode(n)

	// Should be marked suspect
	if m1.nodeMap["127.0.0.103"].State != StateSuspect {
		t.Fatalf("Expect node to be suspect")
	}

	// Should increment seqno
	if m2.sequenceNum != 1 {
		t.Fatalf("bad seqno %v", m2.sequenceNum)
	}
	if m3.sequenceNum != 1 {
		t.Fatalf("bad seqno %v", m3.sequenceNum)
	}
}

func TestMemberList_ProbeNode(t *testing.T) {
	m1 := HostMemberlist("127.0.0.200", t)
	m1.config.RTT = time.Millisecond
	m1.config.Interval = 10 * time.Millisecond
	m2 := HostMemberlist("127.0.0.201", t)

	a1 := alive{Node: "127.0.0.200", Addr: []byte{127, 0, 0, 200}, Incarnation: 1}
	m1.aliveNode(&a1)
	a2 := alive{Node: "127.0.0.201", Addr: []byte{127, 0, 0, 201}, Incarnation: 1}
	m1.aliveNode(&a2)

	n := m1.nodeMap["127.0.0.201"]
	m1.probeNode(n)

	// Should be marked suspect
	if n.State != StateAlive {
		t.Fatalf("Expect node to be alive")
	}

	// Should increment seqno
	if m1.sequenceNum != 1 {
		t.Fatalf("bad seqno %v", m2.sequenceNum)
	}
}

func TestMemberList_ResetNodes(t *testing.T) {
	m := GetMemberlist(t)
	a1 := alive{Node: "test1", Addr: []byte{127, 0, 0, 1}, Incarnation: 1}
	m.aliveNode(&a1)
	a2 := alive{Node: "test2", Addr: []byte{127, 0, 0, 2}, Incarnation: 1}
	m.aliveNode(&a2)
	a3 := alive{Node: "test3", Addr: []byte{127, 0, 0, 3}, Incarnation: 1}
	m.aliveNode(&a3)
	d := dead{Node: "test2", Incarnation: 1}
	m.deadNode(&d)

	m.resetNodes()
	if len(m.nodes) != 2 {
		t.Fatalf("Bad length")
	}
	if _, ok := m.nodeMap["test2"]; ok {
		t.Fatalf("test2 should be unmapped")
	}
}

func TestMemberList_NextSeq(t *testing.T) {
	m := &Memberlist{}
	if m.nextSeqNo() != 1 {
		t.Fatalf("bad sequence no")
	}
	if m.nextSeqNo() != 2 {
		t.Fatalf("bad sequence no")
	}
}

func TestMemberList_SetAckChannel(t *testing.T) {
	m := &Memberlist{ackHandlers: make(map[uint32]*ackHandler)}

	ch := make(chan bool, 1)
	m.setAckChannel(0, ch, 10*time.Millisecond)

	if _, ok := m.ackHandlers[0]; !ok {
		t.Fatalf("missing handler")
	}
	time.Sleep(11 * time.Millisecond)

	if _, ok := m.ackHandlers[0]; ok {
		t.Fatalf("non-reaped handler")
	}
}

func TestMemberList_SetAckHandler(t *testing.T) {
	m := &Memberlist{ackHandlers: make(map[uint32]*ackHandler)}

	f := func() {}
	m.setAckHandler(0, f, 10*time.Millisecond)

	if _, ok := m.ackHandlers[0]; !ok {
		t.Fatalf("missing handler")
	}
	time.Sleep(11 * time.Millisecond)

	if _, ok := m.ackHandlers[0]; ok {
		t.Fatalf("non-reaped handler")
	}
}

func TestMemberList_InvokeAckHandler(t *testing.T) {
	m := &Memberlist{ackHandlers: make(map[uint32]*ackHandler)}

	// Does nothing
	m.invokeAckHandler(0)

	var b bool
	f := func() { b = true }
	m.setAckHandler(0, f, 10*time.Millisecond)

	// Should set b
	m.invokeAckHandler(0)
	if !b {
		t.Fatalf("b not set")
	}

	if _, ok := m.ackHandlers[0]; ok {
		t.Fatalf("non-reaped handler")
	}
}

func TestMemberList_InvokeAckHandler_Channel(t *testing.T) {
	m := &Memberlist{ackHandlers: make(map[uint32]*ackHandler)}

	// Does nothing
	m.invokeAckHandler(0)

	ch := make(chan bool, 1)
	m.setAckChannel(0, ch, 10*time.Millisecond)

	// Should send message
	m.invokeAckHandler(0)

	select {
	case v := <-ch:
		if v != true {
			t.Fatalf("Bad value")
		}
	default:
		t.Fatalf("message not sent")
	}

	if _, ok := m.ackHandlers[0]; ok {
		t.Fatalf("non-reaped handler")
	}
}

func TestMemberList_AliveNode_NewNode(t *testing.T) {
	ch := make(chan *Node, 1)
	m := GetMemberlist(t)
	m.NotifyJoin(ch)

	a := alive{Node: "test", Addr: []byte{127, 0, 0, 1}, Incarnation: 1}
	m.aliveNode(&a)

	if len(m.nodes) != 1 {
		t.Fatalf("should add node")
	}

	state, ok := m.nodeMap["test"]
	if !ok {
		t.Fatalf("should map node")
	}

	if state.Incarnation != 1 {
		t.Fatalf("bad incarnation")
	}
	if state.State != StateAlive {
		t.Fatalf("bad state")
	}
	if time.Now().Sub(state.StateChange) > time.Second {
		t.Fatalf("bad change delta")
	}

	// Check for a join message
	select {
	case join := <-ch:
		if join.Name != "test" {
			t.Fatalf("bad node name")
		}
	default:
		t.Fatalf("no join message")
	}
}

func TestMemberList_AliveNode_SuspectNode(t *testing.T) {
	ch := make(chan *Node, 1)
	m := GetMemberlist(t)

	a := alive{Node: "test", Addr: []byte{127, 0, 0, 1}, Incarnation: 1}
	m.aliveNode(&a)

	// Listen only after first join
	m.NotifyJoin(ch)

	// Make suspect
	state := m.nodeMap["test"]
	state.State = StateSuspect
	state.StateChange = state.StateChange.Add(-time.Hour)

	// Old incarnation number, should not change
	m.aliveNode(&a)
	if state.State != StateSuspect {
		t.Fatalf("update with old incarnation!")
	}

	// Should reset to alive now
	a.Incarnation = 2
	m.aliveNode(&a)
	if state.State != StateAlive {
		t.Fatalf("no update with new incarnation!")
	}

	if time.Now().Sub(state.StateChange) > time.Second {
		t.Fatalf("bad change delta")
	}

	// Check for a no join message
	select {
	case <-ch:
		t.Fatalf("got bad join message")
	default:
	}
}

func TestMemberList_AliveNode_Idempotent(t *testing.T) {
	ch := make(chan *Node, 1)
	m := GetMemberlist(t)

	a := alive{Node: "test", Addr: []byte{127, 0, 0, 1}, Incarnation: 1}
	m.aliveNode(&a)

	// Listen only after first join
	m.NotifyJoin(ch)

	// Make suspect
	state := m.nodeMap["test"]
	stateTime := state.StateChange

	// Should reset to alive now
	a.Incarnation = 2
	m.aliveNode(&a)
	if state.State != StateAlive {
		t.Fatalf("non idempotent")
	}

	if stateTime != state.StateChange {
		t.Fatalf("should not change state")
	}

	// Check for a no join message
	select {
	case <-ch:
		t.Fatalf("got bad join message")
	default:
	}
}

func TestMemberList_SuspectNode_NoNode(t *testing.T) {
	m := GetMemberlist(t)
	s := suspect{Node: "test", Incarnation: 1}
	m.suspectNode(&s)
	if len(m.nodes) != 0 {
		t.Fatalf("don't expect nodes")
	}
}

func TestMemberList_SuspectNode(t *testing.T) {
	m := GetMemberlist(t)
	m.config.Interval = time.Millisecond
	m.config.SuspicionMult = 1
	a := alive{Node: "test", Addr: []byte{127, 0, 0, 1}, Incarnation: 1}
	m.aliveNode(&a)

	state := m.nodeMap["test"]
	state.StateChange = state.StateChange.Add(-time.Hour)

	s := suspect{Node: "test", Incarnation: 1}
	m.suspectNode(&s)

	if state.State != StateSuspect {
		t.Fatalf("Bad state")
	}

	change := state.StateChange
	if time.Now().Sub(change) > time.Second {
		t.Fatalf("bad change delta")
	}

	// Wait for the timeout
	time.Sleep(10 * time.Millisecond)

	if state.State != StateDead {
		t.Fatalf("Bad state")
	}

	if time.Now().Sub(state.StateChange) > time.Second {
		t.Fatalf("bad change delta")
	}
	if !state.StateChange.After(change) {
		t.Fatalf("should increment time")
	}
}

func TestMemberList_SuspectNode_DoubleSuspect(t *testing.T) {
	m := GetMemberlist(t)
	a := alive{Node: "test", Addr: []byte{127, 0, 0, 1}, Incarnation: 1}
	m.aliveNode(&a)

	state := m.nodeMap["test"]
	state.StateChange = state.StateChange.Add(-time.Hour)

	s := suspect{Node: "test", Incarnation: 1}
	m.suspectNode(&s)

	if state.State != StateSuspect {
		t.Fatalf("Bad state")
	}

	change := state.StateChange
	if time.Now().Sub(change) > time.Second {
		t.Fatalf("bad change delta")
	}

	// Suspect again
	m.suspectNode(&s)

	if state.StateChange != change {
		t.Fatalf("unexpected state change")
	}
}

func TestMemberList_SuspectNode_OldSuspect(t *testing.T) {
	m := GetMemberlist(t)
	a := alive{Node: "test", Addr: []byte{127, 0, 0, 1}, Incarnation: 10}
	m.aliveNode(&a)

	state := m.nodeMap["test"]
	state.StateChange = state.StateChange.Add(-time.Hour)

	s := suspect{Node: "test", Incarnation: 1}
	m.suspectNode(&s)

	if state.State != StateAlive {
		t.Fatalf("Bad state")
	}
}

func TestMemberList_DeadNode_NoNode(t *testing.T) {
	m := GetMemberlist(t)
	d := dead{Node: "test", Incarnation: 1}
	m.deadNode(&d)
	if len(m.nodes) != 0 {
		t.Fatalf("don't expect nodes")
	}
}

func TestMemberList_DeadNode(t *testing.T) {
	ch := make(chan *Node, 1)
	m := GetMemberlist(t)
	m.NotifyLeave(ch)
	a := alive{Node: "test", Addr: []byte{127, 0, 0, 1}, Incarnation: 1}
	m.aliveNode(&a)

	state := m.nodeMap["test"]
	state.StateChange = state.StateChange.Add(-time.Hour)

	d := dead{Node: "test", Incarnation: 1}
	m.deadNode(&d)

	if state.State != StateDead {
		t.Fatalf("Bad state")
	}

	change := state.StateChange
	if time.Now().Sub(change) > time.Second {
		t.Fatalf("bad change delta")
	}

	select {
	case leave := <-ch:
		if leave.Name != "test" {
			t.Fatalf("bad node name")
		}
	default:
		t.Fatalf("no leave message")
	}
}

func TestMemberList_DeadNode_Double(t *testing.T) {
	ch := make(chan *Node, 1)
	m := GetMemberlist(t)
	a := alive{Node: "test", Addr: []byte{127, 0, 0, 1}, Incarnation: 1}
	m.aliveNode(&a)

	state := m.nodeMap["test"]
	state.StateChange = state.StateChange.Add(-time.Hour)

	d := dead{Node: "test", Incarnation: 1}
	m.deadNode(&d)

	// Notify after the first dead
	m.NotifyLeave(ch)

	// Should do nothing
	d.Incarnation = 2
	m.deadNode(&d)

	select {
	case <-ch:
		t.Fatalf("should not get leave")
	default:
	}
}

func TestMemberList_DeadNode_OldDead(t *testing.T) {
	m := GetMemberlist(t)
	a := alive{Node: "test", Addr: []byte{127, 0, 0, 1}, Incarnation: 10}
	m.aliveNode(&a)

	state := m.nodeMap["test"]
	state.StateChange = state.StateChange.Add(-time.Hour)

	d := dead{Node: "test", Incarnation: 1}
	m.deadNode(&d)

	if state.State != StateAlive {
		t.Fatalf("Bad state")
	}
}

func TestMemberList_MergeState(t *testing.T) {
	m := GetMemberlist(t)
	a1 := alive{Node: "test1", Addr: []byte{127, 0, 0, 1}, Incarnation: 1}
	m.aliveNode(&a1)
	a2 := alive{Node: "test2", Addr: []byte{127, 0, 0, 2}, Incarnation: 1}
	m.aliveNode(&a2)
	a3 := alive{Node: "test3", Addr: []byte{127, 0, 0, 3}, Incarnation: 1}
	m.aliveNode(&a3)

	s := suspect{Node: "test1", Incarnation: 1}
	m.suspectNode(&s)

	remote := []pushNodeState{
		pushNodeState{
			Name:        "test1",
			Addr:        []byte{127, 0, 0, 1},
			Incarnation: 2,
			State:       StateAlive,
		},
		pushNodeState{
			Name:        "test2",
			Addr:        []byte{127, 0, 0, 2},
			Incarnation: 1,
			State:       StateSuspect,
		},
		pushNodeState{
			Name:        "test3",
			Addr:        []byte{127, 0, 0, 3},
			Incarnation: 1,
			State:       StateDead,
		},
		pushNodeState{
			Name:        "test4",
			Addr:        []byte{127, 0, 0, 4},
			Incarnation: 2,
			State:       StateAlive,
		},
	}

	// Listen for changes
	joinCh := make(chan *Node, 1)
	leaveCh := make(chan *Node, 1)
	m.NotifyJoin(joinCh)
	m.NotifyLeave(leaveCh)

	// Merge remote state
	m.mergeState(remote)

	// Check the states
	state := m.nodeMap["test1"]
	if state.State != StateAlive || state.Incarnation != 2 {
		t.Fatalf("Bad state %v", state)
	}

	state = m.nodeMap["test2"]
	if state.State != StateSuspect || state.Incarnation != 1 {
		t.Fatalf("Bad state %v", state)
	}

	state = m.nodeMap["test3"]
	if state.State != StateDead || state.Incarnation != 1 {
		t.Fatalf("Bad state %v", state)
	}

	state = m.nodeMap["test4"]
	if state.State != StateAlive || state.Incarnation != 2 {
		t.Fatalf("Bad state %v", state)
	}

	// Check the channels
	select {
	case j := <-joinCh:
		if j.Name != "test4" {
			t.Fatalf("bad node %v", j)
		}
	default:
		t.Fatalf("Expect join")
	}

	select {
	case l := <-leaveCh:
		if l.Name != "test3" {
			t.Fatalf("bad node %v", l)
		}
	default:
		t.Fatalf("Expect leave")
	}
}