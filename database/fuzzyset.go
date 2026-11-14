package database

import (
	"math"
	"slices"
	"strings"
	"unicode"
)

// fuzzySet finds the entries closest to a word. It ports the fuzzyset
// Haskell package (0.2), which PostgREST uses for the hints of its errors
// (Error.hs, Data.FuzzySet), with the defaults of PostgREST: grams of
// size 3, then 2, and a Levenshtein score on the best 50 entries.
type fuzzySet struct {
	exact     map[string]string      // lowercase entry -> entry
	items     map[int][]fuzzyItem    // per gram size, the entries
	matchDict map[string][]gramCount // gram -> entries that contain it
}

type fuzzyItem struct {
	norm  float64 // Euclidean norm of the gram counts
	entry string  // lowercase entry
}

type gramCount struct {
	index int // index in items of the gram size
	count int
}

const (
	fuzzyGramLower = 2
	fuzzyGramUpper = 3
)

// newFuzzySet returns a set of the entries.
func newFuzzySet(entries []string) *fuzzySet {
	fs := &fuzzySet{
		exact:     map[string]string{},
		items:     map[int][]fuzzyItem{},
		matchDict: map[string][]gramCount{},
	}
	for _, e := range entries {
		fs.add(e)
	}
	return fs
}

func (fs *fuzzySet) add(value string) {
	key := strings.ToLower(value)
	if _, ok := fs.exact[key]; ok {
		return
	}
	fs.exact[key] = value
	for size := fuzzyGramLower; size <= fuzzyGramUpper; size++ {
		counts := gramCounts(key, size)
		index := len(fs.items[size])
		sum := 0
		for g, c := range counts {
			sum += c * c
			fs.matchDict[g] = append(fs.matchDict[g], gramCount{index, c})
		}
		fs.items[size] = append(fs.items[size], fuzzyItem{math.Sqrt(float64(sum)), key})
	}
}

// fuzzyMatch is an entry and its score, from 0 to 1.
type fuzzyMatch struct {
	score float64
	entry string
}

// get returns the entries with a score of at least minScore, the best
// first. An entry equal to value, ignoring case, is the only match.
func (fs *fuzzySet) get(value string, minScore float64) []fuzzyMatch {
	key := strings.ToLower(value)
	if e, ok := fs.exact[key]; ok {
		return []fuzzyMatch{{1, e}}
	}
	for size := fuzzyGramUpper; size >= fuzzyGramLower; size-- {
		if m := fs.matches(key, minScore, size); len(m) > 0 {
			return m
		}
	}
	return nil
}

// getOne returns the best entry with a score of at least minScore, and
// false when there is none.
func (fs *fuzzySet) getOne(value string, minScore float64) (string, bool) {
	if m := fs.get(value, minScore); len(m) > 0 {
		return m[0].entry, true
	}
	return "", false
}

func (fs *fuzzySet) matches(key string, minScore float64, size int) []fuzzyMatch {
	counts := gramCounts(key, size)
	scores := map[int]int{}
	sum := 0
	for g, c := range counts {
		sum += c * c
		for _, gc := range fs.matchDict[g] {
			scores[gc.index] += c * gc.count
		}
	}
	if len(scores) == 0 {
		return nil
	}
	norm := math.Sqrt(float64(sum))
	items := fs.items[size]
	indexes := make([]int, 0, len(scores))
	for i := range scores {
		indexes = append(indexes, i)
	}
	slices.Sort(indexes)
	results := make([]fuzzyMatch, len(indexes))
	for i, index := range indexes {
		results[i] = fuzzyMatch{float64(scores[index]) / (norm * items[index].norm), items[index].entry}
	}
	byScore := func(a, b fuzzyMatch) int {
		switch {
		case a.score > b.score:
			return -1
		case a.score < b.score:
			return 1
		}
		return 0
	}
	slices.SortStableFunc(results, byScore)
	// the Levenshtein score of the best 50 entries replaces the cosine one
	results = results[:min(50, len(results))]
	for i := range results {
		results[i].score = levenshteinScore(results[i].entry, key)
	}
	slices.SortStableFunc(results, byScore)
	var found []fuzzyMatch
	for _, r := range results {
		if r.score >= minScore {
			found = append(found, fuzzyMatch{r.score, fs.exact[r.entry]})
		}
	}
	return found
}

// gramCounts counts the grams of a size in a value. The value keeps only
// its letters, digits, spaces and commas, and a hyphen pads it on each side.
func gramCounts(value string, size int) map[string]int {
	var b strings.Builder
	b.WriteByte('-')
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) || r == ',' {
			b.WriteRune(r)
		}
	}
	b.WriteByte('-')
	runes := []rune(b.String())
	counts := map[string]int{}
	for i := 0; i+size <= len(runes); i++ {
		counts[string(runes[i:i+size])]++
	}
	return counts
}

// levenshteinScore is 1 minus the Levenshtein distance of a and b over the
// length of the longer one.
func levenshteinScore(a, b string) float64 {
	ra, rb := []rune(a), []rune(b)
	longer := max(len(ra), len(rb))
	if longer == 0 {
		return 1
	}
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return 1 - float64(prev[len(rb)])/float64(longer)
}
