package chaincode

import (
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	abci "github.com/tendermint/tendermint/abci/types"
)

// EventHandler defines the interface of handling the events of application
type EventHandler interface {
	Handle(ctx contractapi.TransactionContextInterface, events []abci.Event) (continue_ bool, err error)
}

// DefaultMultiEventHandler returns a handler which satisfies minimum requirements
func DefaultMultiEventHandler() MultiEventHandler {
	return NewMultiEventHandler(
		EventHandlerFunc(HandlePacketEvent),
		EventHandlerFunc(HandlePacketAcknowledgementEvent),
	)
}

// EventHandlerFunc is a function type which handles application events
type EventHandlerFunc func(ctx contractapi.TransactionContextInterface, events []abci.Event) (continue_ bool, err error)

// Handle implements EventHandler.Handle
func (f EventHandlerFunc) Handle(ctx contractapi.TransactionContextInterface, events []abci.Event) (continue_ bool, err error) {
	return f(ctx, events)
}

// MultiEventHandler is a handler which handles application events by multiple handlers
type MultiEventHandler struct {
	hs []EventHandler
}

// NewMultiEventHandler creates a instance of MultiEventHandler
func NewMultiEventHandler(handlers ...EventHandler) MultiEventHandler {
	return MultiEventHandler{hs: handlers}
}

// Handle handles a given events by multiple handlers
func (mh MultiEventHandler) Handle(ctx contractapi.TransactionContextInterface, events []abci.Event) error {
	for _, h := range mh.hs {
		continue_, err := h.Handle(ctx, events)
		if err != nil {
			return err
		}
		if !continue_ {
			return nil
		}
	}
	return nil
}
