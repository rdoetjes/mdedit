package spell

import (
	"strings"
	"unicode"
)

// Runes that, besides letters and digits, can be part of a chunk of text
// between spaces and Markdown punctuation: a word, a URL, a file name.
const chunkPunct = "_'’-./:@+#%~=*^"

// Runes trimmed from the ends of a chunk: emphasis markers, full stops and
// colons, quotes and dashes.
const edgePunct = "_*~.:'’-"

// Runes that, inside a chunk, make it something other than prose: a URL,
// e-mail address, file name or identifier.
const notProse = "_./:@+#%=*^~"

func isChunkRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune(chunkPunct, r)
}

// chunk checks the chunk of text starting at i and returns its end.
func (c *checker) chunk(i, to int) int {
	j := i
	for j < to && isChunkRune(c.text[j]) {
		j++
	}
	c.checkChunk(i, j)
	return j
}

// checkChunk checks the words of a chunk, split at hyphens, unless it isn't
// prose.
func (c *checker) checkChunk(from, to int) {
	from, to = c.trim(from, to, edgePunct)
	for i := from; i < to; i++ {
		if r := c.text[i]; unicode.IsDigit(r) || strings.ContainsRune(notProse, r) {
			return
		}
	}
	start := from
	for i := from; i <= to; i++ {
		if i == to || c.text[i] == '-' {
			c.checkWord(c.trim(start, i, "'’"))
			start = i + 1
		}
	}
}

// trim returns [from, to) without the runes of cut at either end.
func (c *checker) trim(from, to int, cut string) (int, int) {
	for from < to && strings.ContainsRune(cut, c.text[from]) {
		from++
	}
	for to > from && strings.ContainsRune(cut, c.text[to-1]) {
		to--
	}
	return from, to
}

// checkWord records text[from:to] if it is a misspelt word.
func (c *checker) checkWord(from, to int) {
	if to-from < 2 {
		return
	}
	w := string(c.text[from:to])
	if isUpper(w) || hasInnerUpper(c.text[from+1:to]) || c.d.Correct(w) {
		return
	}
	c.misses = append(c.misses, Miss{Start: from, End: to, Word: w})
}

func hasInnerUpper(rs []rune) bool {
	for _, r := range rs {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}
