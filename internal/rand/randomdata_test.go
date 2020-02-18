package rand

import "testing"

func TestRandLetterBytes(t *testing.T) {
	name := randLetterBytes(20)
	t.Logf("%v", string(name))
}

/*
func benchmarkRandLetterBytes1(b *testing.B, size int) {
	for n := 0; n < b.N; n++ {
		_ = origRandBytes(size)
	}
}

func Benchmark1RandLetterBytes20(b *testing.B)      { benchmarkRandLetterBytes1(b, 20) }
func Benchmark1RandLetterBytes100(b *testing.B)     { benchmarkRandLetterBytes1(b, 100) }
func Benchmark1RandLetterBytes500(b *testing.B)     { benchmarkRandLetterBytes1(b, 500) }
func Benchmark1RandLetterBytes1000(b *testing.B)    { benchmarkRandLetterBytes1(b, 1000) }
func Benchmark1RandLetterBytes1000000(b *testing.B) { benchmarkRandLetterBytes1(b, 1000000) }
*/

func benchmarkRandBytes2(b *testing.B, size int) {
	for n := 0; n < b.N; n++ {
		_ = randBytes(size)
	}
}

func Benchmark2RandBytes20(b *testing.B)      { benchmarkRandBytes2(b, 20) }
func Benchmark2RandBytes100(b *testing.B)     { benchmarkRandBytes2(b, 100) }
func Benchmark2RandBytes500(b *testing.B)     { benchmarkRandBytes2(b, 500) }
func Benchmark2RandBytes1000(b *testing.B)    { benchmarkRandBytes2(b, 1000) }
func Benchmark2RandBytes1000000(b *testing.B) { benchmarkRandBytes2(b, 1000000) }

func benchmarkRandString2(b *testing.B, size int) {
	for n := 0; n < b.N; n++ {
		_ = randString(size)
	}
}

func Benchmark2RandString20(b *testing.B)      { benchmarkRandString2(b, 20) }
func Benchmark2RandString100(b *testing.B)     { benchmarkRandString2(b, 100) }
func Benchmark2RandString500(b *testing.B)     { benchmarkRandString2(b, 500) }
func Benchmark2RandString1000(b *testing.B)    { benchmarkRandString2(b, 1000) }
func Benchmark2RandString1000000(b *testing.B) { benchmarkRandString2(b, 1000000) }

func benchmarkRandLetterBytes2(b *testing.B, size int) {
	for n := 0; n < b.N; n++ {
		_ = randLetterBytes(size)
	}
}

func Benchmark2RandLetterBytes20(b *testing.B)      { benchmarkRandLetterBytes2(b, 20) }
func Benchmark2RandLetterBytes100(b *testing.B)     { benchmarkRandLetterBytes2(b, 100) }
func Benchmark2RandLetterBytes500(b *testing.B)     { benchmarkRandLetterBytes2(b, 500) }
func Benchmark2RandLetterBytes1000(b *testing.B)    { benchmarkRandLetterBytes2(b, 1000) }
func Benchmark2RandLetterBytes1000000(b *testing.B) { benchmarkRandLetterBytes2(b, 1000000) }

/*
func benchmarkRandBytes3(b *testing.B, size int) {
	for n := 0; n < b.N; n++ {
		_ = chanRandBytes(size)
	}
}

func Benchmark3RandBytes20(b *testing.B)      { benchmarkRandBytes3(b, 20) }
func Benchmark3RandBytes100(b *testing.B)     { benchmarkRandBytes3(b, 100) }
func Benchmark3RandBytes500(b *testing.B)     { benchmarkRandBytes3(b, 500) }
func Benchmark3RandBytes1000(b *testing.B)    { benchmarkRandBytes3(b, 1000) }
func Benchmark3RandBytes1000000(b *testing.B) { benchmarkRandBytes3(b, 1000000) }

func benchmarkRandLetterBytes3(b *testing.B, size int) {
	for n := 0; n < b.N; n++ {
		_ = chanRandLetterBytes(size)
	}
}

func Benchmark3RandLetterBytes20(b *testing.B)      { benchmarkRandLetterBytes3(b, 20) }
func Benchmark3RandLetterBytes100(b *testing.B)     { benchmarkRandLetterBytes3(b, 100) }
func Benchmark3RandLetterBytes500(b *testing.B)     { benchmarkRandLetterBytes3(b, 500) }
func Benchmark3RandLetterBytes1000(b *testing.B)    { benchmarkRandLetterBytes3(b, 1000) }
func Benchmark3RandLetterBytes1000000(b *testing.B) { benchmarkRandLetterBytes3(b, 1000000) }
*/
