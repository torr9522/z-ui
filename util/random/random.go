package random

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	mathrand "math/rand"
	"time"
)

var numSeq [10]rune
var lowerSeq [26]rune
var upperSeq [26]rune
var numLowerSeq [36]rune
var numUpperSeq [36]rune
var allSeq [62]rune

func init() {
	mathrand.Seed(time.Now().UnixNano())

	for i := 0; i < 10; i++ {
		numSeq[i] = rune('0' + i)
	}
	for i := 0; i < 26; i++ {
		lowerSeq[i] = rune('a' + i)
		upperSeq[i] = rune('A' + i)
	}

	copy(numLowerSeq[:], numSeq[:])
	copy(numLowerSeq[len(numSeq):], lowerSeq[:])

	copy(numUpperSeq[:], numSeq[:])
	copy(numUpperSeq[len(numSeq):], upperSeq[:])

	copy(allSeq[:], numSeq[:])
	copy(allSeq[len(numSeq):], lowerSeq[:])
	copy(allSeq[len(numSeq)+len(lowerSeq):], upperSeq[:])
}

func Seq(n int) string {
	runes := make([]rune, n)
	for i := 0; i < n; i++ {
		runes[i] = allSeq[mathrand.Intn(len(allSeq))]
	}
	return string(runes)
}

func SecureSeq(n int) string {
	if n <= 0 {
		return ""
	}
	runes := make([]rune, n)
	buf := make([]byte, n*8)
	if _, err := cryptorand.Read(buf); err != nil {
		return Seq(n)
	}
	for i := 0; i < n; i++ {
		value := binary.LittleEndian.Uint64(buf[i*8 : (i+1)*8])
		runes[i] = allSeq[int(value%uint64(len(allSeq)))]
	}
	return string(runes)
}

func SecureInt(min int, max int) int {
	if max <= min {
		return min
	}
	var buf [8]byte
	if _, err := cryptorand.Read(buf[:]); err != nil {
		return min + mathrand.Intn(max-min)
	}
	value := binary.LittleEndian.Uint64(buf[:])
	return min + int(value%uint64(max-min))
}
