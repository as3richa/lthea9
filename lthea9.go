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
	pos int
}

type bigramEntry struct {
	str int
	pos [2]int
}

type SubseqIndex struct {
	buffer []byte
	strs   []bufferString

	monogram [distinctChars][]monogramEntry

	bigram [distinctChars * distinctChars][]bigramEntry
}

type queryResult struct {
	str int
	pos []byte
}

type QueryResult struct {
	str string
	pos []byte
}

const (
	distinctChars   = (0x7e - 0x20 + 1) - 26
	distinctBigrams = distinctChars * distinctChars
	badId           = math.MaxInt32
)

func charId(b byte) int {
	if b <= 0x19 || b >= 0x7f {
		return 0xff
	} else if 0x61 <= b && b <= 0x7a {
		return int(b) - 0x40
	} else {
		return int(b) - 0x20
	}
}

func bigramId(b0, b1 byte) int {
	ch0 := charId(b0)
	ch1 := charId(b1)
	if ch0 == badId || ch1 == badId {
		return badId
	}
	return charId(b0)*distinctChars + ch1
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
		return bytes.Compare(index.strBytes(str1), index.strBytes(str2)) == -1
	})

	{
		k := 1
		for i := 1; i < len(index.strs); i += 1 {
			if bytes.Equal(index.strBytes(i), index.strBytes(i-1)) {
				continue
			}
			index.strs[k] = index.strs[i]
			k += 1
		}
		index.strs = index.strs[:k]
	}

	for str := range index.strs {
		var monogramBitmap [(distinctChars + 7) / 8]byte

		for pos, b := range index.strBytes(str) {
			id := charId(b)
			if id == badId || bitmapTestAndSet(monogramBitmap[:], id) {
				continue
			}

			index.monogram[id] = append(index.monogram[id], monogramEntry{
				str: str,
				pos: pos,
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
		strBytes := index.strBytes(str)

		for pos0, b0 := range strBytes {
			if charId(b0) == 0xff {
				continue
			}

			for pos1, b1 := range strBytes[pos0+1:] {
				id := bigramId(b0, b1)
				if id == badId || bitmapTestAndSet(bigramBitmap[:], id) {
					continue
				}

				index.bigram[id] = append(index.bigram[id], bigramEntry{
					str: str,
					pos: [2]int{pos0, pos1},
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
				str: string(index.strBytes(str)),
				pos: nil,
			})
		}
		return
	}

	if len(bytes) == 1 {
		ary := index.monogram[charId(bytes[0])]
		if maxResults > len(ary) {
			maxResults = len(ary)
		}
		for _, entry := range ary {
			onResult(QueryResult{
				str: string(index.strBytes(entry.str)),
				pos: []byte{toByteSaturating(entry.pos)},
			})
		}
		return
	}

	index.queryBigram(bytes, onResult)
}

func (index *SubseqIndex) queryBigram(bytes []byte, onResult func(QueryResult)) {
}

func (index *SubseqIndex) planLeadingBigram(subseqBytes []byte) (int, func(int, func(QueryResult))) {
	leadingBigram := bigramId(subseqBytes[0], subseqBytes[1])
	ary := index.bigram[leadingBigram]

	execute := func(maxResults int, onResult func(QueryResult)) {
		if maxResults == 0 {
			return
		}

		groupLeadingBigramPos := [2]int{-1, -1}
		var group []queryResult

		emit := func() {
			index.sortAndEmitGroup(group, &maxResults, []byte{byte(groupLeadingBigramPos[0]), byte(groupLeadingBigramPos[1])}, onResult)
		}

		for _, entry := range ary {
			if entry.pos != groupLeadingBigramPos {
				emit()
				if maxResults == 0 {
					return
				}
				group = group[:0]
				groupLeadingBigramPos = entry.pos
			}

			pos := index.matchStr(entry.str, entry.pos[1]+1, subseqBytes[2:])
			if pos == nil {
				continue
			}

			group = append(group, queryResult{
				str: entry.str,
				pos: pos,
			})
		}

		emit()
	}

	return len(ary), execute
}

func (index *SubseqIndex) planLeadingChar(subseqBytes []byte) (int, func(int, func(QueryResult))) {
	ary := index.bigram[bigramId(subseqBytes[0], subseqBytes[1])]
	for i := 2; i < len(subseqBytes); i += 1 {
		candidate := index.bigram[bigramId(subseqBytes[0], subseqBytes[i])]
		if len(candidate) < len(ary) {
			ary = candidate
		}
	}

	execute := func(maxResults int, onResult func(QueryResult)) {
		if maxResults == 0 {
			return
		}

		groupLeadingCharPos := -1
		var group []queryResult

		emit := func() {
			index.sortAndEmitGroup(group, &maxResults, []byte{toByteSaturating(groupLeadingCharPos)}, onResult)
		}

		for _, entry := range ary {
			if entry.pos[0] != groupLeadingCharPos {
				emit()
				if maxResults == 0 {
					return
				}
				group = group[:0]
				groupLeadingCharPos = entry.pos[0]
			}

			pos := index.matchStr(entry.str, entry.pos[0]+1, subseqBytes[1:])
			if pos == nil {
				continue
			}

			group = append(group, queryResult{
				str: entry.str,
				pos: pos,
			})
		}

		emit()
	}

	return len(ary), execute
}

func (index *SubseqIndex) planUnsorted(subseqBytes []byte) (int, func(int, func(QueryResult))) {
	seekLimit := 16
	if len(subseqBytes) < seekLimit {
		seekLimit = len(subseqBytes)
	}

	ary := index.bigram[bigramId(subseqBytes[0], subseqBytes[1])]
	for i := 0; i < seekLimit-1; i += 1 {
		for j := i + 1; j < seekLimit; j += 1 {
			candidate := index.bigram[bigramId(subseqBytes[i], subseqBytes[j])]
			if len(candidate) < len(ary) {
				ary = candidate
			}
		}
	}

	execute := func(maxResults int, onResult func(QueryResult)) {
		var filtered []queryResult

		for _, entry := range ary {
			pos := index.matchStr(entry.str, 0, subseqBytes)
			if pos == nil {
				continue
			}
			filtered = append(filtered, queryResult{
				str: entry.str,
				pos: pos,
			})
		}

		index.sortAndEmitGroup(filtered, &maxResults, nil, onResult)
	}

	return len(ary), execute
}

func (index *SubseqIndex) sortAndEmitGroup(group []queryResult, maxResults *int, leadingPos []byte, onResult func(QueryResult)) {
	sort.Slice(group, func(i, j int) bool {
		cmp := bytes.Compare(group[i].pos, group[j].pos)
		return cmp == -1 || (cmp == 0 && group[i].str < group[j].str)
	})

	if len(group) > *maxResults {
		group = group[:*maxResults]
		*maxResults = 0
	} else {
		*maxResults -= len(group)
	}

	for _, res := range group {
		onResult(QueryResult{
			str: string(index.strBytes(res.str)),
			pos: append(leadingPos, res.pos...),
		})
	}
}

func (index *SubseqIndex) matchStr(str int, startingAt int, subseqBytes []byte) []byte {
	strBytes := index.strBytes(str)
	pos := make([]byte, len(subseqBytes))

	cursor := startingAt
	for i, b := range subseqBytes {
		for cursor < len(strBytes) {
			if strBytes[cursor] == b {
				break
			}
			cursor += 1
		}
		if cursor == len(strBytes) {
			return nil
		}
		pos[i] = toByteSaturating(cursor)
		cursor += 1
	}

	return pos
}

func (index *SubseqIndex) strBytes(str int) []byte {
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
