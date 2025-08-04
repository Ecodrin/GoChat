package utility

import (
	"fmt"
	"encoding/hex"
)

func ToHex(data [32]byte) string {
    return hex.EncodeToString(data[:])
}

func FromHex(hexString string) ([32]byte, error) {
    var result [32]byte
    
    data, err := hex.DecodeString(hexString)
    if err != nil {
        return result, fmt.Errorf("invalid hex string: %v", err)
    }
    
    if len(data) != 32 {
        return result, fmt.Errorf("invalid data length: expected 32, got %d", len(data))
    }
    
    copy(result[:], data)
    return result, nil
}
