package analyzer

import (
	"optimizer/common"
)

type InvocationCount map[common.CallKey]int

func (icnt InvocationCount) GetCount(from, to string) int {
	return icnt[common.CallKey{From: from, To: to}]
}

func (icnt InvocationCount) Exist(from, to string) bool {
	if count, ok := icnt[common.CallKey{From: from, To: to}]; ok && (count != 0) {
		return true
	}
	return false
}

func (icnt InvocationCount) Copy() InvocationCount {
	newMap := make(InvocationCount, len(icnt))

	for key, value := range icnt {
		newMap[key] = value
	}
	return newMap
}

// newNumI_t[common.CallKey{From: node.From, To: node.To}] -= IC.NumIC_t_IC
func (icnt InvocationCount) Decrease(key common.CallKey, val int) {
	icnt[key] -= val
	if icnt[key] <= 0 {
		delete(icnt, key)
	}
}
