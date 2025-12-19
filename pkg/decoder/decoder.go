package decoder

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
)

// ABIWrapper wraps the decoding logic using go-ethereum's ABI parser.
type ABIWrapper struct {
	parsedABI abi.ABI
}

// NewFromJSON creates a decoder from a JSON ABI string
func NewFromJSON(jsonStr string) (*ABIWrapper, error) {
	parsed, err := abi.JSON(strings.NewReader(jsonStr))
	if err != nil {
		return nil, err
	}
	return &ABIWrapper{parsedABI: parsed}, nil
}

// DecodedLog contains parsed human-readable data from a transaction log.
type DecodedLog struct {
	Name   string                 // Event name (e.g., Transfer)
	Inputs map[string]interface{} // Parameter key-value pairs (e.g., from: 0x..., value: 100)
}

// Decode parses a single Log
func (w *ABIWrapper) Decode(log types.Log) (*DecodedLog, error) {
	if len(log.Topics) == 0 {
		return nil, fmt.Errorf("log has no topics")
	}

	// 1. Find the Event definition in ABI based on Topic[0] (Event Signature)
	event, err := w.parsedABI.EventByID(log.Topics[0])
	if err != nil {
		return nil, fmt.Errorf("event signature not found in ABI")
	}

	result := &DecodedLog{
		Name:   event.Name,
		Inputs: make(map[string]interface{}),
	}

	// 2. Parse Data (non-indexed parameters)
	if len(log.Data) > 0 {
		if err := w.parsedABI.UnpackIntoMap(result.Inputs, event.Name, log.Data); err != nil {
			return nil, err
		}
	}

	// 3. Parse Topics (indexed parameters)
	// Iterate through arguments to find indexed ones
	var indexedArgs abi.Arguments
	for _, arg := range event.Inputs {
		if arg.Indexed {
			indexedArgs = append(indexedArgs, arg)
		}
	}

	// Validate topics count (Topics[0] is signature, subsequent ones are indexed parameters)
	if len(log.Topics)-1 != len(indexedArgs) {
		return nil, fmt.Errorf("topic count mismatch: expected %d, got %d", len(indexedArgs), len(log.Topics)-1)
	}

	// Parse indexed parameters one by one
	if err := abi.ParseTopicsIntoMap(result.Inputs, indexedArgs, log.Topics[1:]); err != nil {
		return nil, err
	}

	return result, nil
}
