// Package curation implements the FluxLens curation engine — the layer that
// transforms raw event streams into curated digests balancing freshness,
// source diversity, and redundancy suppression.
//
// The six selection strategies implemented here generalize the algorithms
// originally proposed in:
//
//	Buthalapalli, Y. & Vanga, S. H. (2025). "Balancing Freshness and Diversity
//	in Social Media Digest Systems."
//
// The original paper applied these strategies to social-media digest
// generation; the same formal objective — maximize freshness, ensure source
// coverage, minimize redundancy — applies directly to industrial event
// curation, which is what FluxLens uses them for.
package curation

import (
	"errors"
	"math/rand"
	"sort"
	"time"

	"github.com/sriharshav1/fluxlens/internal/canonical"
)

// Strategy enumerates the six selection strategies. Numbering matches the
// original paper's algorithm catalog.
type Strategy int

const (
	StrategyLatest                       Strategy = 1
	StrategyLatestPerSource              Strategy = 2
	StrategyHybridLatestAndPerSource     Strategy = 3
	StrategyGuaranteedMinDiversity       Strategy = 4
	StrategyGuaranteedMinDiversityRandom Strategy = 5
	StrategyPreferredSources             Strategy = 6
)

// Request encapsulates a single curation call.
type Request struct {
	Strategy         Strategy
	Events           []canonical.Event
	K                int                 // target digest size
	DiversityPercent float64             // 0-100, used by strategies 4-6
	PreferredSources []string            // used by strategy 6
	SuppressionSet   map[string]struct{} // EventIDs to exclude (already shown recently)
	Now              time.Time           // reference time; defaults to time.Now() if zero
	Rand             *rand.Rand          // used by strategy 5; defaults to a fresh PRNG if nil
}

// Result is the output of a curation call.
type Result struct {
	Selected        []canonical.Event
	Strategy        Strategy
	FreshnessScore  float64 // [0, 1]; higher = fresher (1 = all most-recent)
	DiversityScore  float64 // [0, 1]; |unique sources(Selected)| / |unique sources(input)|
	RedundancyScore float64 // [0, 1]; fraction of Selected whose IDs were in SuppressionSet pre-filter
}

// Select dispatches to the requested strategy.
func Select(req Request) (Result, error) {
	if req.K < 0 {
		return Result{}, errors.New("curation: K must be non-negative")
	}
	if req.Now.IsZero() {
		req.Now = time.Now()
	}

	// Apply redundancy pre-filter: remove events whose IDs are in SuppressionSet.
	// The redundancy score is computed below from the count removed.
	prefiltered, removed := filterSuppressed(req.Events, req.SuppressionSet)

	var selected []canonical.Event
	switch req.Strategy {
	case StrategyLatest:
		selected = selectLatest(prefiltered, req.K)
	case StrategyLatestPerSource:
		selected = selectLatestPerSource(prefiltered, req.K)
	case StrategyHybridLatestAndPerSource:
		selected = selectHybrid(prefiltered, req.K)
	case StrategyGuaranteedMinDiversity:
		selected = selectGuaranteedDiversity(prefiltered, req.K, req.DiversityPercent)
	case StrategyGuaranteedMinDiversityRandom:
		rng := req.Rand
		if rng == nil {
			rng = rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec // deterministic alternative is fine for curation
		}
		selected = selectGuaranteedDiversityRandom(prefiltered, req.K, req.DiversityPercent, rng)
	case StrategyPreferredSources:
		selected = selectPreferredSources(prefiltered, req.K, req.DiversityPercent, req.PreferredSources)
	default:
		return Result{}, errors.New("curation: unknown strategy")
	}

	// Sort the final selection by timestamp descending for stable consumer experience.
	sortByTimestampDesc(selected)

	return Result{
		Selected:        selected,
		Strategy:        req.Strategy,
		FreshnessScore:  computeFreshness(selected, req.Events, req.Now),
		DiversityScore:  computeDiversity(selected, req.Events),
		RedundancyScore: redundancyRatio(removed, len(req.Events)),
	}, nil
}

func filterSuppressed(events []canonical.Event, suppress map[string]struct{}) (kept []canonical.Event, removed int) {
	if len(suppress) == 0 {
		return events, 0
	}
	kept = make([]canonical.Event, 0, len(events))
	for _, e := range events {
		if _, hit := suppress[e.EventID]; hit {
			removed++
			continue
		}
		kept = append(kept, e)
	}
	return kept, removed
}

func sortByTimestampDesc(events []canonical.Event) {
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].Timestamp.After(events[j].Timestamp)
	})
}

// selectLatest implements Algorithm 1 from the paper:
// pure freshness — return the K most recent events regardless of source.
func selectLatest(events []canonical.Event, k int) []canonical.Event {
	sorted := append([]canonical.Event(nil), events...)
	sortByTimestampDesc(sorted)
	if k > len(sorted) {
		k = len(sorted)
	}
	return sorted[:k]
}

// selectLatestPerSource implements Algorithm 2:
// pure diversity — return the latest event from each source, up to K total.
// If K < |sources|, oldest-by-source-latest are dropped.
func selectLatestPerSource(events []canonical.Event, k int) []canonical.Event {
	bySource := latestPerSource(events)
	out := make([]canonical.Event, 0, len(bySource))
	for _, e := range bySource {
		out = append(out, e)
	}
	sortByTimestampDesc(out)
	if k < len(out) {
		out = out[:k]
	}
	return out
}

// selectHybrid implements Algorithm 3:
// 1 latest event per source, then fill with latest globally up to K.
func selectHybrid(events []canonical.Event, k int) []canonical.Event {
	bySource := latestPerSource(events)
	selected := make(map[string]struct{}, len(bySource))
	out := make([]canonical.Event, 0, k)
	for _, e := range bySource {
		out = append(out, e)
		selected[e.EventID] = struct{}{}
		if len(out) >= k {
			break
		}
	}
	if len(out) >= k {
		return out[:k]
	}
	sorted := append([]canonical.Event(nil), events...)
	sortByTimestampDesc(sorted)
	for _, e := range sorted {
		if len(out) >= k {
			break
		}
		if _, dup := selected[e.EventID]; dup {
			continue
		}
		out = append(out, e)
		selected[e.EventID] = struct{}{}
	}
	return out
}

// selectGuaranteedDiversity implements Algorithm 4:
// guarantee that at least diversityPercent of sources are represented,
// then fill remaining slots with the globally freshest events.
//
// Concretely: take the latest event from each source; drop the oldest
// floor(|sources|*(1 - diversity/100)) of those source-latest events;
// fill remaining capacity with globally freshest events not already
// selected.
func selectGuaranteedDiversity(events []canonical.Event, k int, diversityPercent float64) []canonical.Event {
	return guaranteedDiversityCore(events, k, diversityPercent, false, nil)
}

// selectGuaranteedDiversityRandom implements Algorithm 5:
// same as Algorithm 4, but evict random source-latest events instead of
// the oldest. Used in the original paper as a baseline to validate that
// Algorithm 4's "drop oldest" rule actually improves average freshness.
func selectGuaranteedDiversityRandom(events []canonical.Event, k int, diversityPercent float64, rng *rand.Rand) []canonical.Event {
	return guaranteedDiversityCore(events, k, diversityPercent, true, rng)
}

// selectPreferredSources implements Algorithm 6:
// run Algorithm 4 first; if no preferred source is represented in the
// result, replace the oldest selected event with the latest event from
// any preferred source that is present in the input.
func selectPreferredSources(events []canonical.Event, k int, diversityPercent float64, preferred []string) []canonical.Event {
	result := selectGuaranteedDiversity(events, k, diversityPercent)
	if len(preferred) == 0 || len(result) == 0 {
		return result
	}
	preferredSet := make(map[string]struct{}, len(preferred))
	for _, p := range preferred {
		preferredSet[p] = struct{}{}
	}
	for _, e := range result {
		if _, ok := preferredSet[e.SourceID]; ok {
			return result // already contains a preferred source
		}
	}
	// Find the latest event from any preferred source in the input.
	var candidate *canonical.Event
	for i := range events {
		if _, ok := preferredSet[events[i].SourceID]; !ok {
			continue
		}
		if candidate == nil || events[i].Timestamp.After(candidate.Timestamp) {
			candidate = &events[i]
		}
	}
	if candidate == nil {
		return result
	}
	// Replace the oldest result entry with the preferred-source candidate.
	oldestIdx := 0
	for i := range result {
		if result[i].Timestamp.Before(result[oldestIdx].Timestamp) {
			oldestIdx = i
		}
	}
	result[oldestIdx] = *candidate
	return result
}

// guaranteedDiversityCore is the shared body of Algorithms 4 and 5.
// When evictRandom is true, oldest-eviction is replaced with random eviction.
func guaranteedDiversityCore(events []canonical.Event, k int, diversityPercent float64, evictRandom bool, rng *rand.Rand) []canonical.Event {
	if diversityPercent < 0 {
		diversityPercent = 0
	}
	if diversityPercent > 100 {
		diversityPercent = 100
	}
	bySource := latestPerSource(events)
	type sl struct {
		src string
		ev  canonical.Event
	}
	ranked := make([]sl, 0, len(bySource))
	for s, e := range bySource {
		ranked = append(ranked, sl{s, e})
	}

	if evictRandom {
		rng.Shuffle(len(ranked), func(i, j int) { ranked[i], ranked[j] = ranked[j], ranked[i] })
	} else {
		// Sort by timestamp ascending so we drop the oldest first.
		sort.SliceStable(ranked, func(i, j int) bool {
			return ranked[i].ev.Timestamp.Before(ranked[j].ev.Timestamp)
		})
	}
	nSources := len(bySource)
	toDrop := int(float64(nSources) * (1 - diversityPercent/100))
	if toDrop < 0 {
		toDrop = 0
	}
	if toDrop > nSources {
		toDrop = nSources
	}

	selected := make(map[string]struct{}, k)
	kept := make([]canonical.Event, 0, k)
	for i := toDrop; i < len(ranked); i++ {
		kept = append(kept, ranked[i].ev)
		selected[ranked[i].ev.EventID] = struct{}{}
		if len(kept) >= k {
			return kept
		}
	}
	sorted := append([]canonical.Event(nil), events...)
	sortByTimestampDesc(sorted)
	for _, e := range sorted {
		if len(kept) >= k {
			break
		}
		if _, dup := selected[e.EventID]; dup {
			continue
		}
		kept = append(kept, e)
		selected[e.EventID] = struct{}{}
	}
	return kept
}

func latestPerSource(events []canonical.Event) map[string]canonical.Event {
	m := make(map[string]canonical.Event)
	for _, e := range events {
		cur, ok := m[e.SourceID]
		if !ok || e.Timestamp.After(cur.Timestamp) {
			m[e.SourceID] = e
		}
	}
	return m
}

func computeFreshness(selected, all []canonical.Event, ref time.Time) float64 {
	if len(selected) == 0 || len(all) == 0 {
		return 0
	}
	var maxAge time.Duration
	for _, e := range all {
		if a := e.AgeAt(ref); a > maxAge {
			maxAge = a
		}
	}
	if maxAge == 0 {
		return 1
	}
	var total float64
	for _, e := range selected {
		fr := 1.0 - float64(e.AgeAt(ref))/float64(maxAge)
		total += fr
	}
	return total / float64(len(selected))
}

func computeDiversity(selected, all []canonical.Event) float64 {
	if len(all) == 0 {
		return 0
	}
	allSources := make(map[string]struct{})
	for _, e := range all {
		allSources[e.SourceID] = struct{}{}
	}
	selSources := make(map[string]struct{})
	for _, e := range selected {
		selSources[e.SourceID] = struct{}{}
	}
	if len(allSources) == 0 {
		return 0
	}
	return float64(len(selSources)) / float64(len(allSources))
}

func redundancyRatio(removed, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(removed) / float64(total)
}
