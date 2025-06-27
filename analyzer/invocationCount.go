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
