package main

import "romanziske/engine"

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
var historyMoves = make([]uint16, 13*maxPly)

func scoreMoves(pos engine.Position, moves *engine.MoveList) {
	ply := int(pos.Ply)

	for index := 0; index < int(moves.Count); index++ {
		move := &moves.Moves[index]

		movingPiece := pos.Squares[move.FromSq()]
		capturedPiece := pos.Squares[move.ToSq()]

		if capturedPiece.Type != engine.NoType {
			score := mvvLva[movingPiece.Type*7+capturedPiece.Type] + mvvLvaOffset
			move.AddScore(score)
		} else if move.Equal(killerMoves[ply]) {
			move.AddScore(mvvLvaOffset - 1000)
		} else if move.Equal(killerMoves[ply+maxPly]) {
			move.AddScore(mvvLvaOffset - 2000)
		} else {
			if pos.SideToMove == 0 {
				move.AddScore(historyMoves[movingPiece.Type*maxPly+capturedPiece.Type+6])
			} else {
				move.AddScore(historyMoves[movingPiece.Type+6*maxPly+capturedPiece.Type])
			}
		}
	}
}
