package ranking

import (
	"sort"

	"github.com/yourusername/nebula/internal/collectors"
)

func Rank(results []collectors.Result) []collectors.Result {
	sort.Slice(results, func(i, j int) bool {
		if results[i].Confidence != results[j].Confidence {
			return results[i].Confidence > results[j].Confidence
		}
		return results[i].FoundAt.After(results[j].FoundAt)
	})
	return results
}
