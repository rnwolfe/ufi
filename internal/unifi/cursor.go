package unifi

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// Cursors are opaque to callers: a base64 wrapper over the upstream integer offset, so the
// wire format stays stable even if pagination changes (spec: Output schema).

func encodeCursor(offset int) string {
	return base64.RawURLEncoding.EncodeToString([]byte("o:" + strconv.Itoa(offset)))
}

func decodeCursor(cursor string) (int, error) {
	if cursor == "" {
		return 0, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, err
	}
	s := string(b)
	if !strings.HasPrefix(s, "o:") {
		return 0, fmt.Errorf("bad cursor")
	}
	n, err := strconv.Atoi(strings.TrimPrefix(s, "o:"))
	if err != nil || n < 0 {
		return 0, fmt.Errorf("bad cursor")
	}
	return n, nil
}
