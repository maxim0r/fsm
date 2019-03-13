//
// General Purpose FSM library.
//
package fsm

import (
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

const (
	NO_INPUT Input = -1
)

// External logger
var log *logrus.Logger

// An Input to give to an FSM.
type Input int

// An Action describes something an FSM will do.
// It returns an Input to allow for automatic chaining of actions.
type Action func(context.Context) (context.Context, Input)

// NO_ACTION is useful for when you need a certain input to just change the state of the FSM without doing anytyhing else.
func NO_ACTION(ctx context.Context) (context.Context, Input) { return ctx, NO_INPUT }

// An Outcome describes the result of running an FSM.
// It describes which state to move to next, and an Action to perform.
type Outcome struct {
	State  int
	Action Action
}

// A State describes one possible state of an FSM.
// It maps Inputs to Outcomes.
type State struct {
	Index    int
	Outcomes map[Input]Outcome
}

// FSM is the main structure defining a Finite State Machine.
type FSM struct {
	sync.Mutex
	states  map[int]State
	current int
}

// InvalidInputError indicates that an input was passed to an FSM which is not valid for its current state.
type InvalidInputError struct {
	StateIndex int
	Input      Input
}

func (err InvalidInputError) Error() string {
	return fmt.Sprintf("input invalid in current state.  (State: %v, Input: %v)", err.StateIndex, err.Input)
}

// ImpossibleStateError indicates that an FSM is in a state which wasn't part of its definition.
// This indicates that either the definition is wrong, or someone is monkeying around with the FSM state manually.
type ImpossibleStateError int

func (err ImpossibleStateError) Error() string {
	return fmt.Sprintf("FSM in impossible state: %d", err)
}

// ClashingStateError indicates that an attempt to define an FSM where two states share the same index was made.
type ClashingStateError int

func (err ClashingStateError) Error() string {
	return fmt.Sprintf("attempt to define FSM with clashing states. Index: %d", err)
}

// Define an FSM from a list of States.
// Will return an  error if you try to use two states with the same index.
func Define(logger *logrus.Logger, states ...State) (*FSM, error) {
	log = logger
	if logger == nil {
		log = logrus.New()
		log.Level = logrus.FatalLevel
	}
	stateMap := map[int]State{}
	for _, s := range states {
		if _, ok := stateMap[s.Index]; ok {
			return nil, ClashingStateError(s.Index)
		}
		stateMap[s.Index] = s
	}

	return &FSM{
		states:  stateMap,
		current: states[0].Index,
	}, nil
}

// Spin the FSM one time.
// This method is thread-safe.
func (f *FSM) Spin(ctx context.Context, in Input) (context.Context, error) {
	f.Lock()
	defer f.Unlock()

	log.Tracef("FSM: get spin input [%d]", in)

	for i := in; i != NO_INPUT; {
		log.Tracef("FSM: process input [%d]", i)

		s, ok := f.states[f.current]
		if !ok {
			log.Tracef("FSM: invalid state [%d]", f.current)
			return ctx, ImpossibleStateError(f.current)
		}

		do, ok := s.Outcomes[i]
		if !ok {
			log.Tracef("FSM: invalid input [%d] in current state [%d]", i, f.current)
			return ctx, InvalidInputError{f.current, i}
		}

		ctx, i = do.Action(ctx)
		f.current = do.State
		log.Tracef("FSM: set current state [%d] with next input [%d]", f.current, i)
	}

	return ctx, nil
}
