package lthea9

import (
	"sort"
	"strings"
)

type bufferString struct {
	start int
	end   int
}

type SubseqIndexBuilder struct {
	buffer strings.Builder
	strs   []bufferString
}

func (builder *SubseqIndexBuilder) Insert(str string) {
	builder.strs = append(builder.strs, bufferString{
		start: builder.buffer.Len(),
		end:   builder.buffer.Len() + len(str),
	})
	builder.buffer.WriteString(str)
}

func (builder *SubseqIndexBuilder) Build() SubseqIndex {
	buffer := builder.buffer.String()
	strs := builder.strs

	lexicographic := make([]int, len(strs))
	for i := 0; i < len(strs); i += 1 {
		lexicographic[i] = i
	}
	sort.Slice(lexicographic, func(i, j int) bool {
		left := buffer[strs[i].start:strs[i].end]
		right := buffer[strs[j].start:strs[j].end]
		return left < right
	})

	return SubseqIndex{
		buffer: buffer,
		strs:   strs,
	}
}

const (
	distinctChars = (0x7e - 0x20 + 1) - 26 + 1
	binaryChar    = distinctChars - 1
)

func asciiByteToChar(b byte) byte {
	if b <= 0x19 {
		return binaryChar
	} else if b <= 0x60 {
		return b - 0x20
	} else if b <= 0x7a {
		return b - 0x40
	} else if b <= 0x7e {
		return b - 0x20
	} else {
		return binaryChar
	}
}

type SubseqIndex struct {
	buffer string
	strs   []bufferString

	lexicographic []int

	monogram [distinctChars][]struct {
		str   int
		index byte
	}

	bigram [distinctChars * distinctChars][]struct {
		str     int
		indices [2]byte
	}
}

type QueryResult struct {
	str     string
	matches []byte
}

func (index *SubseqIndex) Query(subseq string) []QueryResult {
	return nil
}
