package sebakcommon

import (
	"math/rand"
	"testing"
	"time"
)

var stringSlice500 []string
var stringMap500 map[string]bool

var stringSlice1000 []string
var stringMap1000 map[string]bool

var stringSlice5000 []string
var stringMap5000 map[string]bool

func init() {
	rand.Seed(time.Now().UnixNano())

	length := 32
	nStrings500 := 500
	nStrings1000 := 1000
	nStrings5000 := 5000

	stringSlice500 = []string{}
	stringMap500 = map[string]bool{}

	for i := 0; i < nStrings500; i++ {
		str := randSeq(length)
		stringSlice500 = append(stringSlice500, str)
		stringMap500[str] = true
	}

	stringSlice1000 = []string{}
	stringMap1000 = map[string]bool{}

	for i := 0; i < nStrings1000; i++ {
		str := randSeq(length)
		stringSlice1000 = append(stringSlice1000, str)
		stringMap1000[str] = true
	}

	stringSlice5000 = []string{}
	stringMap5000 = map[string]bool{}

	for i := 0; i < nStrings5000; i++ {
		str := randSeq(length)
		stringSlice5000 = append(stringSlice5000, str)
		stringMap5000[str] = true
	}
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func BenchmarkFoundInStringSlice500(b *testing.B) {
	for n := 0; n < b.N; n++ {
		InStringArray(stringSlice500, stringSlice500[rand.Intn(len(stringSlice500))])
	}
}

func BenchmarkFoundInStringMap500(b *testing.B) {
	for n := 0; n < b.N; n++ {
		InStringMap(stringMap500, stringSlice500[rand.Intn(len(stringSlice500))])
	}
}

func BenchmarkFoundInStringSlice1000(b *testing.B) {
	for n := 0; n < b.N; n++ {
		InStringArray(stringSlice1000, stringSlice1000[rand.Intn(len(stringSlice1000))])
	}
}

func BenchmarkFoundInStringMap1000(b *testing.B) {
	for n := 0; n < b.N; n++ {
		InStringMap(stringMap1000, stringSlice1000[rand.Intn(len(stringSlice1000))])
	}
}

func BenchmarkFoundInStringSlice5000(b *testing.B) {
	for n := 0; n < b.N; n++ {
		InStringArray(stringSlice5000, stringSlice5000[rand.Intn(len(stringSlice5000))])
	}
}

func BenchmarkFoundInStringMap5000(b *testing.B) {
	for n := 0; n < b.N; n++ {
		InStringMap(stringMap5000, stringSlice5000[rand.Intn(len(stringSlice5000))])
	}
}

func BenchmarkEqualityOrderedSlice500(b *testing.B) {
	for n := 0; n < b.N; n++ {
		if !IsStringArrayEqual(stringSlice500, stringSlice500) {
			b.FailNow()
		}
	}
}

func BenchmarkEqualityMap500(b *testing.B) {
	for n := 0; n < b.N; n++ {
		if !IsStringMapEqual(stringMap500, stringMap500) {
			b.FailNow()
		}
	}
}

func BenchmarkEqualityMapWithHash500(b *testing.B) {
	for n := 0; n < b.N; n++ {
		if !IsStringMapEqualWithHash(stringMap500, stringMap500) {
			b.FailNow()
		}
	}
}

func BenchmarkEqualityOrderedSlice1000(b *testing.B) {
	for n := 0; n < b.N; n++ {
		if !IsStringArrayEqual(stringSlice1000, stringSlice1000) {
			b.FailNow()
		}
	}
}

func BenchmarkEqualityMap1000(b *testing.B) {
	for n := 0; n < b.N; n++ {
		if !IsStringMapEqual(stringMap1000, stringMap1000) {
			b.FailNow()
		}
	}
}

func BenchmarkEqualityMapWithHash1000(b *testing.B) {
	for n := 0; n < b.N; n++ {
		if !IsStringMapEqualWithHash(stringMap1000, stringMap1000) {
			b.FailNow()
		}
	}
}

func BenchmarkEqualityOrderedSlice5000(b *testing.B) {
	for n := 0; n < b.N; n++ {
		if !IsStringArrayEqual(stringSlice5000, stringSlice5000) {
			b.FailNow()
		}
	}
}

func BenchmarkEqualityMap5000(b *testing.B) {
	for n := 0; n < b.N; n++ {
		if !IsStringMapEqual(stringMap5000, stringMap5000) {
			b.FailNow()
		}
	}
}

func BenchmarkEqualityMapWithHash5000(b *testing.B) {
	for n := 0; n < b.N; n++ {
		if !IsStringMapEqualWithHash(stringMap5000, stringMap5000) {
			b.FailNow()
		}
	}
}
