package fsm

import (
	"context"
	"testing"
)

const (
	test_input_1 = iota
	test_input_2
	test_input_3
)

const (
	test_state_1 = iota
	test_state_2
	test_state_3
)

func assertState(t *testing.T, ctx context.Context, fsm *FSM, i Input, s int) {
	oldState := fsm.current

	ctx, err := fsm.Spin(ctx, i)
	if err != nil {
		t.Fatal(err.Error())
		return
	}

	if fsm.current != s {
		t.Errorf("FSM in wrong state. (Input: %v, State: %v -> %v)", i, oldState, s)
	}
}

func TestStates(t *testing.T) {
	ctx := context.Background()

	state1 := State{
		Index: test_state_1,
		Outcomes: map[Input]Outcome{
			test_input_1: Outcome{test_state_2, NO_ACTION},
			test_input_2: Outcome{test_state_3, NO_ACTION},
			test_input_3: Outcome{test_state_1, NO_ACTION},
		},
	}
	state2 := State{
		Index: test_state_2,
		Outcomes: map[Input]Outcome{
			test_input_1: Outcome{test_state_1, NO_ACTION},
			test_input_2: Outcome{test_state_1, NO_ACTION},
			test_input_3: Outcome{test_state_1, NO_ACTION},
		},
	}
	state3 := State{
		Index: test_state_3,
		Outcomes: map[Input]Outcome{
			test_input_1: Outcome{test_state_2, NO_ACTION},
			test_input_2: Outcome{test_state_1, NO_ACTION},
			test_input_3: Outcome{test_state_1, NO_ACTION},
		},
	}

	fsm, err := Define(state1, state2, state3)
	if err != nil {
		t.Fatal("Failed to define FSM: ", err)
	}

	t.Log("1: 1 -> 2")
	assertState(t, ctx, fsm, test_input_1, test_state_2)

	t.Log("3: 2 -> 1")
	assertState(t, ctx, fsm, test_input_3, test_state_1)

	t.Log("2: 1 -> 3")
	assertState(t, ctx, fsm, test_input_2, test_state_3)

	t.Log("1: 3 -> 2")
	assertState(t, ctx, fsm, test_input_1, test_state_2)

	t.Log("2: 2 -> 1")
	assertState(t, ctx, fsm, test_input_2, test_state_1)

	t.Log("3: 1 -> 1")
	assertState(t, ctx, fsm, test_input_3, test_state_1)
}

// Test actions which return next inputs.
func TestChain(t *testing.T) {
	ctx := context.Background()

	func1_hit := false
	func2_hit := false
	func3_hit := false
	state1 := State{
		Index: test_state_1,
		Outcomes: map[Input]Outcome{
			test_input_1: Outcome{test_state_2,
				func(ctx context.Context) (context.Context, Input) { func1_hit = true; return ctx, test_input_2 }},
			test_input_2: Outcome{test_state_3,
				func(ctx context.Context) (context.Context, Input) { func3_hit = true; return ctx, NO_INPUT }},
			test_input_3: Outcome{test_state_1, NO_ACTION},
		},
	}
	state2 := State{
		Index: test_state_2,
		Outcomes: map[Input]Outcome{
			test_input_1: Outcome{test_state_1, NO_ACTION},
			test_input_2: Outcome{test_state_1,
				func(ctx context.Context) (context.Context, Input) { func2_hit = true; return ctx, test_input_2 }},
			test_input_3: Outcome{test_state_1, NO_ACTION},
		},
	}
	state3 := State{
		Index: test_state_3,
		Outcomes: map[Input]Outcome{
			test_input_1: Outcome{test_state_2, NO_ACTION},
			test_input_2: Outcome{test_state_1, NO_ACTION},
			test_input_3: Outcome{test_state_1, NO_ACTION},
		},
	}

	fsm, err := Define(state1, state2, state3)
	if err != nil {
		t.Fatal("Failed to define FSM: ", err)
	}

	// Spin with input 1.
	// The state chain should go 1 -> 2 -> 1 -> 3 and all 3 functions should have been hit.
	assertState(t, ctx, fsm, test_input_1, test_state_3)
	if !(func1_hit && func2_hit && func3_hit) {
		t.Errorf("Didn't hit all 3 actions. 1: %v, 2: %v, 3: %v.", func1_hit, func2_hit, func3_hit)
	}
}

func TestImpossibleState(t *testing.T) {
	ctx := context.Background()

	state1 := State{
		Index: test_state_1,
		Outcomes: map[Input]Outcome{
			test_input_1: Outcome{test_state_2, NO_ACTION},
			test_input_2: Outcome{test_state_3, NO_ACTION},
		},
	}
	state2 := State{
		Index: test_state_2,
		Outcomes: map[Input]Outcome{
			test_input_1: Outcome{test_state_1, NO_ACTION},
			test_input_2: Outcome{test_state_1, NO_ACTION},
		},
	}

	fsm, err := Define(state1, state2)
	if err != nil {
		t.Fatal("Failed to define FSM: ", err)
	}

	// Put the FSM in an impossible state.
	fsm.current = test_state_3
	ctx, err = fsm.Spin(ctx, test_input_1)
	if err == nil {
		t.Fatalf("FSM didn't error when spun in impossible state.")
	}
	switch err.(type) {
	case ImpossibleStateError:
		t.Logf("FSM corrently returned error: %v", err.Error())
	default:
		t.Fatalf("FSM returned wrong error type: %T", err)
	}
}

func TestInvalidInput(t *testing.T) {
	ctx := context.Background()

	state1 := State{
		Index: test_state_1,
		Outcomes: map[Input]Outcome{
			test_input_1: Outcome{test_state_2, NO_ACTION},
			test_input_2: Outcome{test_state_3, NO_ACTION},
		},
	}
	state2 := State{
		Index: test_state_2,
		Outcomes: map[Input]Outcome{
			test_input_1: Outcome{test_state_1, NO_ACTION},
			test_input_2: Outcome{test_state_1, NO_ACTION},
		},
	}

	fsm, err := Define(state1, state2)
	if err != nil {
		t.Fatal("Failed to define FSM: ", err)
	}

	// Spin with invalid input value.
	ctx, err = fsm.Spin(ctx, test_input_3)
	if err == nil {
		t.Fatalf("FSM didn't error when spun with invalid input.")
	}
	switch err.(type) {
	case InvalidInputError:
		t.Logf("FSM corrently returned error: %v", err.Error())
	default:
		t.Fatalf("FSM returned wrong error type: %T", err)
	}
}

// Test that we error if you try to create an FSM with clashing states.
func TestStateClash(t *testing.T) {
	// Define two states with the same index.
	state1 := State{
		Index: test_state_1,
		Outcomes: map[Input]Outcome{
			test_input_1: Outcome{test_state_2, NO_ACTION},
			test_input_2: Outcome{test_state_3, NO_ACTION},
		},
	}
	state2 := State{
		Index: test_state_1,
		Outcomes: map[Input]Outcome{
			test_input_1: Outcome{test_state_1, NO_ACTION},
			test_input_2: Outcome{test_state_1, NO_ACTION},
		},
	}

	// We should error on this define.
	_, err := Define(state1, state2)

	if err == nil {
		t.Fatalf("Didn't error creating FSM with clashing states.")
	}
	switch err.(type) {
	case ClashingStateError:
		t.Logf("FSM corrently returned error: %v", err.Error())
	default:
		t.Fatalf("FSM returned wrong error type: %T", err)
	}
}
