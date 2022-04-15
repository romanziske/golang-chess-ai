package main

import (
	"romanziske/engine"
)

var mvvLva = [...]uint16{
	105, 205, 305, 405, 505, 605,
	104, 204, 304, 404, 504, 604,
	103, 203, 303, 403, 503, 603,
	102, 202, 302, 402, 502, 602,
	101, 201, 301, 401, 501, 601,
	100, 200, 300, 400, 500, 600,
	0, 0, 0, 0, 0, 0}

var mvvLvaOffset uint16 = 10000

var killerMoves = make([]engine.Move, maxPly*maxKillerMoves)
var historyMoves = make([]int, 13*maxPly)

func ScoreMoves(pos engine.Position, moves *engine.MoveList) {
	ply := pos.Ply

	for index := 0; index < int(moves.Count); index++ {
		move := &moves.Moves[index]

		movingPiece := pos.Squares[move.FromSq()]
		capturedPiece := pos.Squares[move.ToSq()]

		if capturedPiece.Type != engine.NoType {
			score := mvvLva[movingPiece.Type*7+capturedPiece.Type] + mvvLvaOffset
			move.AddScore(score)
		} else if move.Equal(killerMoves[ply]) {
			move.AddScore(mvvLvaOffset - 1000)
		} else if move.Equal(killerMoves[int(ply)+maxPly]) {
			move.AddScore(mvvLvaOffset - 2000)
		} else {
			if pos.SideToMove == 0 {
				move.AddScore(uint16(historyMoves[movingPiece.Type*uint8(maxPly)+move.ToSq()]))
			} else {
				move.AddScore(uint16(historyMoves[(movingPiece.Type+6)*uint8(maxPly)+move.ToSq()]))
			}
		}
	}
}

func SortMoves(currIndex int, moves *engine.MoveList) {
	bestIndex := currIndex
	bestScore := moves.Moves[bestIndex].Score()

	for index := bestIndex; index < int(moves.Count); index++ {
		if moves.Moves[index].Score() > bestScore {
			bestIndex = index
			bestScore = moves.Moves[index].Score()
		}
	}

	tempMove := moves.Moves[currIndex]
	moves.Moves[currIndex] = moves.Moves[bestIndex]
	moves.Moves[bestIndex] = tempMove
}

func StoreKillerMove(pos engine.Position, move engine.Move) {
	ply := pos.Ply

	killerMoves[maxPly+int(ply)] = killerMoves[ply]
	killerMoves[ply] = move
}
