package main

import "math"

type Elo struct {
	k float64
}

type EloDict struct {
	Player Player
	Rank   int
}

type ByRank []*EloDict

func (a ByRank) Len() int      { return len(a) }
func (a ByRank) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByRank) Less(i, j int) bool {
	if a[i].Rank != a[j].Rank {
		return a[i].Rank > a[j].Rank
	}
	return a[i].Player.Nickname < a[j].Player.Nickname
}

func NewEloDict(p Player) *EloDict {
	return &EloDict{Rank: 1000, Player: p}
}

func (e *Elo) getExpected(a, b int) float64 {
	return float64(1) / (1 + math.Pow(10, float64((b-a))/400))
}

func pow(a, b int) int {
	p := 1
	for b > 0 {
		if b&1 != 0 {
			p *= a
		}
		b >>= 1
		a *= a
	}
	return p
}

func round(f float64) int {
	return int(math.Floor(f + .5))
}

func (e *Elo) updateRating(expected float64, actual float64, current int) int {
	return round(float64(current) + e.k*(actual-expected))
}
