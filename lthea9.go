package lthea9

import (
	"bytes"
	"math"
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

type monogramEntry struct {
	str int
	pos byte
}

type bigramEntry struct {
	str int
	pos [2]byte
}

type SubseqIndex struct {
	buffer []byte
	strs   []bufferString

	monogram [distinctChars][]monogramEntry

	bigram [distinctChars * distinctChars][]bigramEntry
}

type QueryResult struct {
	str string
	pos []byte
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

func bigramId(ch0, ch1 byte) int {
	return int(ch0)*distinctChars + int(ch1)
}

func (builder *SubseqIndexBuilder) Insert(str string) {
	builder.strs = append(builder.strs, bufferString{
		start: builder.buffer.Len(),
		end:   builder.buffer.Len() + len(str),
	})
	builder.buffer.WriteString(str)
}

func (builder *SubseqIndexBuilder) Build() SubseqIndex {
	if len(builder.strs) == 0 {
		return SubseqIndex{}
	}

	index := SubseqIndex{
		buffer: []byte(builder.buffer.String()),
		strs:   builder.strs,
	}

	sort.Slice(index.strs, func(str1, str2 int) bool {
		return bytes.Compare(index.stringBytes(str1), index.stringBytes(str2)) == -1
	})

	{
		k := 1
		for i := 1; i < len(index.strs); i += 1 {
			if bytes.Equal(index.stringBytes(i), index.stringBytes(i-1)) {
				continue
			}
			index.strs[k] = index.strs[i]
			k += 1
		}
		index.strs = index.strs[:k]
	}

	for str := range index.strs {
		var monogramBitmap [(distinctChars + 7) / 8]byte

		for pos, b := range index.stringBytes(str) {
			ch := asciiByteToChar(b)
			if bitmapTestAndSet(monogramBitmap[:], int(ch)) {
				continue
			}

			index.monogram[ch] = append(index.monogram[ch], monogramEntry{
				str: str,
				pos: toByteSaturating(pos),
			})
		}
	}
	for _, ary := range index.monogram {
		sort.Slice(ary, func(i, j int) bool {
			return ary[i].pos < ary[j].pos || (ary[i].pos == ary[j].pos && ary[i].str < ary[j].str)
		})
	}

	for str := range index.strs {
		var bigramBitmap [(distinctChars*distinctChars + 7) / 8]byte
		strBytes := index.stringBytes(str)

		for pos0, b0 := range strBytes {
			ch := asciiByteToChar(b0)
			for pos1, b1 := range strBytes[pos0+1:] {
				id := bigramId(ch, asciiByteToChar(b1))
				if bitmapTestAndSet(bigramBitmap[:], id) {
					continue
				}

				index.bigram[id] = append(index.bigram[id], bigramEntry{
					str: str,
					pos: [2]byte{
						toByteSaturating(pos0),
						toByteSaturating(pos1),
					},
				})
			}
		}
	}
	for _, ary := range index.bigram {
		sort.Slice(ary, func(i, j int) bool {
			return ary[i].pos[0] < ary[j].pos[0] ||
				(ary[i].pos[0] == ary[j].pos[0] && ary[i].pos[1] < ary[j].pos[1]) ||
				(ary[i].pos == ary[j].pos && ary[i].str < ary[j].str)
		})
	}

	return index
}

func (index *SubseqIndex) Query(subseq string, maxResults int, onResult func(QueryResult)) {
	bytes := []byte(subseq)

	if len(bytes) == 0 {
		if maxResults > len(index.strs) {
			maxResults = len(index.strs)
		}
		for str := 0; str < maxResults; str += 1 {
			onResult(QueryResult{
				str: string(index.stringBytes(str)),
				pos: nil,
			})
		}
		return
	}

	if len(bytes) == 1 {
		ary := index.monogram[asciiByteToChar(bytes[0])]
		if maxResults > len(ary) {
			maxResults = len(ary)
		}
		for _, entry := range ary {
			onResult(QueryResult{
				str: string(index.stringBytes(entry.str)),
				pos: []byte{entry.pos},
			})
		}
		return
	}

	index.queryBigram(bytes, onResult)
}

func (index *SubseqIndex) queryBigram(bytes []byte, onResult func(QueryResult)) {
	leadingBigramFactor := 1
	leadingCharFactor := 4
	unsortedFactor := 16

	chars := make([]byte, len(bytes))
	for i, b := range bytes {
		chars[i] = asciiByteToChar(b)
	}

	leadingBigram := bigramId(chars[0], chars[1])
	leadingBigramCost := leadingBigramFactor * len(index.bigram[leadingBigram])

	var bestLeadingCharBigram int
	bestLeadingCharCost := math.MaxInt32
	{
		for i := 2; i < len(chars); i += 1 {
			bigram := bigramId(chars[0], chars[i])
			cost := len(index.bigram[bigram])
			if cost < bestLeadingCharCost {
				bestLeadingCharBigram = bigram
				bestLeadingCharCost = cost
			}
		}
		bestLeadingCharCost *= leadingCharFactor
	}

	var bestUnsortedBigram int
	bestUnsortedBigramCost := math.MaxInt32
	{
		unsortedSeekLimit := 16
		if unsortedSeekLimit > len(chars) {
			unsortedSeekLimit = len(chars)
		}

		for i := 2; i < unsortedSeekLimit-1; i += 1 {
			for j := i + 1; j < unsortedSeekLimit; j += 1 {
				bigram := bigramId(chars[i], chars[j])
				cost := len(index.bigram[bigram])
				if cost < bestUnsortedBigramCost {
					bestUnsortedBigram = bigram
					bestUnsortedBigramCost = cost
				}
			}
		}
		bestUnsortedBigramCost *= unsortedFactor
	}

	if leadingBigramCost <= bestLeadingCharCost && leadingBigramCost <= bestUnsortedBigramCost {
		_ = leadingBigram
	} else if bestLeadingCharCost <= leadingBigramCost && bestLeadingCharCost <= bestUnsortedBigramCost {
		_ = bestLeadingCharBigram
	} else {
		_ = bestUnsortedBigram
	}
}

func (index *SubseqIndex) stringBytes(str int) []byte {
	return index.buffer[index.strs[str].start:index.strs[str].end]
}

func bitmapTestAndSet(bitmap []byte, k int) bool {
	if (bitmap[k/8]>>uint(k%8))&1 == 1 {
		return true
	}
	bitmap[k/8] |= 1 << uint(k%8)
	return false
}

func toByteSaturating(n int) byte {
	if n <= 0xff {
		return byte(n)
	}
	return 0xff
}
