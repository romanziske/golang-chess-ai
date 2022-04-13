package main

/*
#cgo LDFLAGS: -L ${SRCDIR}/NNUE/lib -l nnueprobe
#include <NNUE/lib/nnue-probe/src/nnue.h>

*/
import "C"

import (
	"fmt"
	"log"
	"romanziske/engine"
	"time"
)

const (
	MAXVALUE int = 1000

	maxPly         uint8 = 200
	maxKillerMoves int   = 2
	maxDepth       int   = 0
	maxQuisceDepth int   = 0
)

var nodes int = 0
var cuts int = 0
var storedMovesUsed int = 0

var tt = NewTranspositionTable()

func main() {
	start := time.Now()
	fmt.Println(search("r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1", 1, 10))
	elapsed := time.Since(start)
	log.Printf("Search took %s", elapsed)
	fmt.Println(len(killerMoves))

}

func search(fenStr string, depth int, time int) engine.Move {

	//load NNUE
	C.nnue_init(C.CString("./NNUE/networks/nn.nnue"))

	//create game
	var pos engine.Position
	pos.LoadFEN(fenStr)

	return iterativeDeepening(pos, depth)
}

func iterativeDeepening(pos engine.Position, depth int) engine.Move {
	var bestMove engine.Move
	for level := 1; level <= depth; level++ {
		fmt.Println(level)
		bestMove = root(pos, level, -MAXVALUE, MAXVALUE)
	}
	return bestMove
}

func root(pos engine.Position, depth int, alpha int, beta int) engine.Move {
	nodes += 1
	bestValue := -MAXVALUE

	moves := engine.GenMoves(&pos)
	bestMove := moves.Moves[0]
	for index := 0; index < int(moves.Count); index++ {
		move := moves.Moves[index]

		if !pos.MakeMove(move) {
			pos.UnmakeMove(move)
			continue
		}

		value := -negamax(pos, depth-1, -beta, -alpha)
		pos.UnmakeMove(move)

		if value > bestValue {
			bestValue = value
			bestMove = move
		}

		if value > alpha {
			alpha = value
		}

		if alpha >= beta {
			cuts += 1
			break
		}
	}

	return bestMove
}

func negamax(pos engine.Position, depth int, alpha int, beta int) int {
	nodes += 1

	tempAlpha := alpha

	ttValue, exist := tt.ReadEntry(pos.Hash, depth, alpha, beta)

	if exist {
		storedMovesUsed += 1
		return ttValue
	}

	if depth == 0 {
		eval := C.nnue_evaluate_fen(C.CString(pos.GenFEN()))
		return int(eval)
	}

	value := -MAXVALUE
	moves := engine.GenMoves(&pos)
	for index := 0; index < int(moves.Count); index++ {
		move := moves.Moves[index]

		if !pos.MakeMove(move) {
			pos.UnmakeMove(move)
			continue
		}

		value = Max(value, -negamax(pos, depth-1, -beta, -alpha))
		pos.UnmakeMove(move)

		if value > alpha {
			alpha = value
		}

		if alpha >= beta {
			cuts += 1
			break
		}
	}

	var flag int
	if value <= tempAlpha {
		flag = ExactFlag
	} else if value >= beta {
		flag = LowerFlag
	} else {
		flag = ExactFlag
	}

	tt.WriteEntry(pos.Hash, depth, flag, value)

	return value
}
