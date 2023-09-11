package persist

import (
	"bytes"
	"fmt"

	"github.com/axiomesh/axiom-kit/types"
)

func CompositeKey(prefix string, buffer bytes.Buffer) []byte {
	return append([]byte(prefix), []byte(fmt.Sprintf("%v", buffer.String()))...)
}

func ComposeStateKey(addr *types.Address, key []byte) []byte {
	return append(addr.Bytes(), key...)
}
