package rand

import (
	"bytes"
	"math/rand"
	"sync"
	"time"
)

// Bytes returns a random slice of bytes
func Bytes(n int) []byte {
	return randBytes(n)
}

// String returns a random string
func String(n int) string {
	return randString(n)
}

// LetterBytes returns a random slice of bytes picked in the [0-9]|[a-z] range
func LetterBytes(n int) []byte {
	return randLetterBytes(n)
}

// LetterString returns a random string picked in the [0-9]|[a-z] range
func LetterString(n int) string {
	return randLetterString(n)
}

/*
func origRandString(n int) string {
	return string(origRandBytes(n))
}

func origRandBytes(n int) []byte {
	// from https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-golang
	const letterBytes = "abcdefghijklmnopqrstuvwxyz0123456789"
	const (
		letterIdxBits = 6                    // 6 bits to represent a letter index
		letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
		letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
	)
	src := rand.NewSource(time.Now().UnixNano())
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}
	return b
}
*/

var (
	onceSource  sync.Once
	rgen        *rand.Rand
	onceLetters sync.Once
	randMutex   sync.Mutex
)

func seed() {
	src := rand.NewSource(time.Now().UnixNano())
	rgen = rand.New(src) // #nosec
}

func randBytes(n int) []byte {
	onceSource.Do(seed)
	buf := make([]byte, n)
	randMutex.Lock() // the mutex doesn't add any significant time - alternative to mutex: singleton w/ goroutine
	_, _ = rgen.Read(buf)
	randMutex.Unlock()
	return buf
}

func randString(n int) string {
	return string(randBytes(n)) // this is not optimal but the cost of this extra copy is only about 10%
}

var letters []byte

func makeLetters() {
	// adds "a" to pad over 256 locations (0-9 U a-z makes up to 252 only and we want to cover the range of uint8)
	// do the "a" is slightly more frequent than other signs. The trade-off here is speed over exact randomness
	letters = bytes.Repeat([]byte("abcdefghijklmnopqrstuvwxyz0123456789a"), 7)
}

func randLetterBytes(n int) []byte {
	onceLetters.Do(makeLetters)
	buf := randBytes(n)
	for i, b := range buf {
		buf[i] = letters[b]
	}
	return buf
}

func randLetterString(n int) string {
	return string(randLetterBytes(n))
}

/*
// a channel based version: it is actually a bit slower - channel overhead and copy offsets
// any gain from buffering ahead a random stream.
const (
	randChunk  = 1024
	randBuffer = 2048
)

var (
	rchan    chan []byte
	onceChan sync.Once
)

func bgseed() {
	seed()
	rchan = make(chan []byte, randBuffer)
	go func(bgrand chan<- []byte) {
		buf := make([]byte, randChunk)
		for {
			_, _ = rgen.Read(buf)
			bgrand <- buf
		}
	}(rchan)
}

func chanRandBytes(n int) []byte {
	onceChan.Do(bgseed)
	buf := make([]byte, ((n/randChunk)+1)*randChunk)
	for i := 0; i < len(buf); i += randChunk {
		chunk := <-rchan
		copy(buf[i:i+randChunk], chunk[0:])
	}
	return buf[0:n]
}

func chanRandString(n int) string {
	return string(chanRandBytes(n))
}

func chanRandLetterBytes(n int) []byte {
	onceLetters.Do(makeLetters)
	buf := chanRandBytes(n)
	for i, b := range buf {
		buf[i] = letters[uint8(b)]
	}
	return buf
}
*/
