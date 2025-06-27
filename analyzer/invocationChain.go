package analyzer

import (
	"fmt"
	"strings"
)

// InvocationChain represents an invocation chain as described in the paper.
// It consists of a sequence of microservices and the minimum occurrence count
// of its invocations within a specific trace.
type InvocationChain struct {
	Microservices []string
	NumIC_t_IC    int // NumIC_i^t_k from the paper, minimum occurrence count of invocations in this chain for a trace t_k
}

// NewInvocationChain creates and returns a new empty InvocationChain.
func NewInvocationChain() *InvocationChain {
	return &InvocationChain{
		Microservices: make([]string, 0),
		NumIC_t_IC:    0,
	}
}

// Append adds a microservice to the end of the invocation chain.
func (ic *InvocationChain) Append(ms string) {
	ic.Microservices = append(ic.Microservices, ms)
}

// GetTail returns the last microservice in the invocation chain.
// It returns an empty string if the chain is empty.
func (ic *InvocationChain) GetTail() string {
	if ic.IsEmpty() {
		return ""
	}
	return ic.Microservices[len(ic.Microservices)-1]
}

// IsEmpty checks if the invocation chain contains no microservices.
func (ic *InvocationChain) IsEmpty() bool {
	return len(ic.Microservices) == 0
}

// Copy creates a new InvocationChain by copying a portion of the original.
// It's used for the "List.copy" operation in Algorithm 1.
// The 'endIndex' is exclusive, meaning the element at endIndex is not included.
func (ic *InvocationChain) Copy(startIndex, endIndex int) *InvocationChain {
	if startIndex < 0 || startIndex > len(ic.Microservices) || endIndex < startIndex || endIndex > len(ic.Microservices) {
		// Handle invalid indices appropriately, for now return an empty chain
		fmt.Printf("Warning: Invalid indices for InvocationChain.Copy - startIndex: %d, endIndex: %d, chain length: %d\n", startIndex, endIndex, len(ic.Microservices))
		return NewInvocationChain()
	}

	newChain := NewInvocationChain()
	newChain.Microservices = make([]string, endIndex-startIndex)
	copy(newChain.Microservices, ic.Microservices[startIndex:endIndex])
	// The NumIC_t_IC count needs to be updated based on the new chain's context
	// For the purpose of "List.copy" as a prefix, the count might be inherited or re-evaluated later.
	// We'll set it to 0 for now and let the algorithm update it.
	newChain.NumIC_t_IC = 0
	return newChain
}

// String provides a string representation of the InvocationChain for map keys.
func (ic *InvocationChain) String() string {
	return strings.Join(ic.Microservices, "->")
}

// InvocationChains holds a collection of unique invocation chains and their total occurrences.
type InvocationChains struct {
	// Use a map where the key is the string representation of an InvocationChain
	// and the value is its total occurrence count across all traces.
	// The paper refers to this as 'IC' (a collection of invocation chains in T).
	Chains map[string]int
}

// NewInvocationChains creates and returns a new empty InvocationChains collection.
func NewInvocationChains() *InvocationChains {
	return &InvocationChains{
		Chains: make(map[string]int),
	}
}

// Append adds an InvocationChain to the collection, accumulating its occurrence count.
// This function assumes the input 'ic' is a single trace's invocation chain with its NumIC_t_IC set.
// It effectively implements NumIC_i^T = sum(NumIC_i^t_k).
func (ics *InvocationChains) Append(ic *InvocationChain) {
	if ic.IsEmpty() {
		return
	}
	ics.Chains[ic.String()] += ic.NumIC_t_IC
}

// Add merges another InvocationChains collection into the current one.
// This is for future usage as you mentioned.
func (ics *InvocationChains) Add(other *InvocationChains) {
	for chainStr, count := range other.Chains {
		ics.Chains[chainStr] += count
	}
}
