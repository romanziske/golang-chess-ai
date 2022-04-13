package main

const (
	ExactFlag int = iota
	UpperFlag
	LowerFlag
)

type TT_Entry struct {
	Hash  uint64
	Depth int
	Value int
	Flag  int
}

type TranspositionTable struct {
	entries [768000]TT_Entry
	size    uint64
}

func NewTranspositionTable() *TranspositionTable {
	const entrieCount = 768000 //64 mb default size

	var entries [entrieCount]TT_Entry

	return &TranspositionTable{entries, entrieCount / 12}
}

func (tt *TranspositionTable) ReadEntry(hash uint64, depth, alpha, beta int) (int, bool) {
	entry := &tt.entries[hash%tt.size]

	if entry.Hash >= hash {
		ttVal := entry.Value
		ttFlag := entry.Flag

		if entry.Depth == depth {
			if ttFlag == ExactFlag {
				return ttVal, true
			} else if ttFlag == LowerFlag {
				alpha = Max(alpha, ttVal)
			} else if ttFlag == UpperFlag {
				beta = Min(beta, ttVal)
			}

			if alpha >= beta {
				return ttVal, true
			}
		}
	}

	return 0, false
}

func (tt *TranspositionTable) WriteEntry(hash uint64, depth, flag, value int) {
	entry := &tt.entries[hash%tt.size]
	entry.Hash = hash
	entry.Depth = depth
	entry.Flag = flag

	entry.Value = value
}

func (tt *TranspositionTable) Clear() {
	var idx uint64
	for idx = 0; idx < tt.size; idx++ {
		tt.entries[idx] = TT_Entry{}
	}
}
