package decoder

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

func TestDecode(t *testing.T) {
	// 1. Define standard ERC20 ABI
	const abiJSON = `[{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]`
	
	// 2. Initialize Decoder
	d, err := NewFromJSON(abiJSON)
	assert.NoError(t, err)

	// 3. Prepare test data
	transferSig := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
	sender := common.HexToAddress("0x1111111111111111111111111111111111111111")
	receiver := common.HexToAddress("0x2222222222222222222222222222222222222222")
	amount := big.NewInt(1000000)

	// 4. Construct Log
	// Indexed parameters go into Topics (Note: Address needs to be padded to 32 bytes)
	// Non-indexed parameters (value) need ABI encoding into Data
	
	parsedABI, _ := abi.JSON(strings.NewReader(abiJSON))
	// Pack only packs non-indexed arguments
	// Transfer(from, to, value) -> only value is non-indexed in inputs
	packedData, err := parsedABI.Events["Transfer"].Inputs.NonIndexed().Pack(amount)
	assert.NoError(t, err)

	log := types.Log{
		Topics: []common.Hash{
			transferSig,
			common.BytesToHash(sender.Bytes()),   // Topic 1: From
			common.BytesToHash(receiver.Bytes()), // Topic 2: To
		},
		Data: packedData,
	}

	// 5. Execute decoding
	decoded, err := d.Decode(log)
	assert.NoError(t, err)
	assert.Equal(t, "Transfer", decoded.Name)
	
	// Verify fields
	assert.Equal(t, sender, decoded.Inputs["from"])
	assert.Equal(t, receiver, decoded.Inputs["to"])
	assert.Equal(t, amount, decoded.Inputs["value"]) // Note: type matching is important; ABI usually decodes to *big.Int
}

func TestNewFromJSON_Fail(t *testing.T) {
	_, err := NewFromJSON("invalid json")
	assert.Error(t, err)
}

func TestDecode_ErrorCases(t *testing.T) {
	const abiJSON = `[{"anonymous":false,"inputs":[],"name":"Empty","type":"event"}]`
	d, _ := NewFromJSON(abiJSON)

	// Case 1: No topics
	_, err := d.Decode(types.Log{Topics: []common.Hash{}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no topics")

	// Case 2: Unknown Event Signature
	unknownSig := crypto.Keccak256Hash([]byte("Unknown()"))
	_, err = d.Decode(types.Log{Topics: []common.Hash{unknownSig}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "signature not found")

	// Case 3: Topic count mismatch
	const abiWithIndexed = `[{"anonymous":false,"inputs":[{"indexed":true,"name":"a","type":"address"}],"name":"Event","type":"event"}]`
	d2, _ := NewFromJSON(abiWithIndexed)
	event, _ := d2.parsedABI.EventByID(crypto.Keccak256Hash([]byte("Event(address)")))
	_, err = d2.Decode(types.Log{Topics: []common.Hash{event.ID}}) // Missing indexed topic
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "topic count mismatch")
}
