package main

/*
#cgo LDFLAGS: -L ${SRCDIR}/NNUE/lib -l nnueprobe
#include <NNUE/lib/nnue-probe/src/nnue.h>

*/
import "C"

import (
	"net/http"
	"romanziske/engine"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	MAXVALUE int = 1000

	maxPly         int = 200
	maxKillerMoves int = 2
	maxDepth       int = 0
	maxQuisceDepth int = 4
)

var nodes int = 0
var cuts int = 0
var storedMovesUsed int = 0

var tt = NewTranspositionTable()

func setupRouter() *gin.Engine {
	r := gin.Default()

	// Get user value
	r.GET("/chess/evaluate", func(c *gin.Context) {
		fenStr, ok := c.GetQuery("fen")

		if !ok {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": "fen parameter is missing",
			})
			return
		}

		maxTimeStr, ok := c.GetQuery("time")

		if !ok {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": "time parameter is missing",
			})
			return
		}

		maxTimeInt, _ := strconv.Atoi(maxTimeStr)

		start := time.Now()
		move := search(fenStr, 6, maxTimeInt)
		elapsed := time.Since(start)
		c.JSON(http.StatusOK, gin.H{
			"bestMove": move.String(),
			"time":     elapsed.String(),
		})
	})

	return r
}

func main() {
	r := setupRouter()
	// Listen and Server in 0.0.0.0:8080
	r.Run(":8080")
}

/* func main() {
	start := time.Now()
	fmt.Println(search("r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 10", 6, 10))
	elapsed := time.Since(start)
	fmt.Printfln("Search took %s", elapsed)
} */

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
		bestMove = root(pos, level, -MAXVALUE, MAXVALUE)
	}
	return bestMove
}

func root(pos engine.Position, depth int, alpha int, beta int) engine.Move {
	nodes += 1
	bestValue := -MAXVALUE

	moves := engine.GenMoves(&pos)
	bestMove := moves.Moves[0]
	ScoreMoves(pos, &moves)
	for index := 0; index < int(moves.Count); index++ {
		SortMoves(index, &moves)
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
		return quiesce(pos, maxQuisceDepth, alpha, beta)
	}

	value := -MAXVALUE
	moves := engine.GenMoves(&pos)
	ScoreMoves(pos, &moves)
	for index := 0; index < int(moves.Count); index++ {
		SortMoves(index, &moves)
		move := moves.Moves[index]

		if !pos.MakeMove(move) {
			pos.UnmakeMove(move)
			continue
		}

		value = Max(value, -negamax(pos, depth-1, -beta, -alpha))
		pos.UnmakeMove(move)

		if value > alpha {
			alpha = value

			if pos.Squares[move.ToSq()].Type == engine.NoType {
				movingPiece := pos.Squares[move.FromSq()]

				if pos.SideToMove == 0 {
					historyMoves[movingPiece.Type*uint8(maxPly)+move.ToSq()] += depth
				} else {
					historyMoves[(movingPiece.Type+6)*uint8(maxPly)+move.ToSq()] += depth
				}
			}
		}

		if alpha >= beta {
			cuts += 1

			if pos.Squares[move.ToSq()].Type == engine.NoType {
				StoreKillerMove(pos, move)
			}

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

func quiesce(pos engine.Position, depth int, alpha int, beta int) int {
	nodes += 1

	eval := int(C.nnue_evaluate_fen(C.CString(pos.GenFEN())))

	if eval >= beta {
		return beta
	}

	if eval > alpha {
		alpha = eval
	}

	if depth == 0 {
		return alpha
	}

	moves := engine.GenMoves(&pos)
	ScoreMoves(pos, &moves)
	for index := 0; index < int(moves.Count); index++ {
		SortMoves(index, &moves)
		move := moves.Moves[index]

		//skip none caputures moves
		if pos.Squares[move.ToSq()].Type == engine.NoType {
			continue
		}

		if !pos.MakeMove(move) {
			pos.UnmakeMove(move)
			continue
		}

		value := -quiesce(pos, depth-1, -beta, -alpha)
		pos.UnmakeMove(move)

		if value > alpha {
			alpha = value

			if value >= beta {
				return beta
			}
		}
	}

	return alpha
}
