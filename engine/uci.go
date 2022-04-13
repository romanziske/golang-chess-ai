package engine

import (
	"bufio"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultBookMoveDelay = 2
)

type UCIInterface struct {
	Search      Search
	OpeningBook map[uint64][]PolyglotEntry

	OptionUseBook       bool
	OptionBookPath      string
	OptionBookMoveDelay int
}

func (inter *UCIInterface) Reset() {
	*inter = UCIInterface{}
}

// Respond to the command "uci"
func (inter *UCIInterface) uciCommandResponse() {
	fmt.Printf("\nid name %v\n", EngineName)
	fmt.Printf("id author %v\n", EngineAuthor)
	fmt.Printf("\noption name Hash type spin default 64 min 1 max 32000\n")
	fmt.Print("option name Clear Hash type button\n")
	fmt.Print("option name Clear History type button\n")
	fmt.Print("option name UseBook type check default false\n")
	fmt.Print("option name BookPath type string default\n")
	fmt.Print("option name BookMoveDelay type spin default 2 min 0 max 10\n")
	fmt.Print("option name MiddleGameContempt type spin default 25 min 0 max 100\n")
	fmt.Print("option name EndGameContempt type spin default 0 min 0 max 100\n")
	fmt.Print("\nAvailable UCI commands:\n")

	fmt.Print("    * uci\n    * isready\n    * ucinewgame")
	fmt.Print("\n    * setoption name <NAME> value <VALUE>")

	fmt.Print("\n    * position")
	fmt.Print("\n\t* fen <FEN>")
	fmt.Print("\n\t* startpos")

	fmt.Print("\n    * go")
	fmt.Print("\n\t* wtime <MILLISECONDS>\n\t* btime <MILLISECONDS>")
	fmt.Print("\n\t* winc <MILLISECONDS>\n\t* binc <MILLISECONDS>")
	fmt.Print("\n\t* movestogo <INTEGER>\n\t* depth <INTEGER>\n\t* nodes <INTEGER>\n\t* movetime <MILLISECONDS>")
	fmt.Print("\n\t* infinite")

	fmt.Print("\n    * stop\n    * quit\n\n")
	fmt.Printf("uciok\n\n")
}

// Respond to the command "position"
func (inter *UCIInterface) positionCommandResponse(command string) {
	// Load in the fen string describing the position,
	// or load in the starting position.
	args := strings.TrimPrefix(command, "position ")
	var fenString string
	if strings.HasPrefix(args, "startpos") {
		args = strings.TrimPrefix(args, "startpos ")
		fenString = FENStartPosition
	} else if strings.HasPrefix(args, "fen") {
		args = strings.TrimPrefix(args, "fen ")
		remaining_args := strings.Fields(args)
		fenString = strings.Join(remaining_args[0:6], " ")
		args = strings.Join(remaining_args[6:], " ")
	}

	// Set the board to the appropriate position and make
	// the moves that have occured if any to update the position.
	inter.Search.Pos.LoadFEN(fenString)
	if strings.HasPrefix(args, "moves") {
		args = strings.TrimSuffix(strings.TrimPrefix(args, "moves"), " ")
		if args != "" {
			for _, moveAsString := range strings.Fields(args) {
				move := MoveFromCoord(&inter.Search.Pos, moveAsString)
				inter.Search.Pos.MakeMove(move)

				// Decrementing the history counter here makes
				// sure that no state is saved on the position's
				// history stack since this move will never be undone.
				inter.Search.Pos.StatePly--
			}
		}
	}
}

// Respond to the command "setoption"
func (inter *UCIInterface) setOptionCommandResponse(command string) {
	fields := strings.Fields(command)
	var option, value string
	parsingWhat := ""

	for _, field := range fields {
		if field == "name" {
			parsingWhat = "name"
		} else if field == "value" {
			parsingWhat = "value"
		} else if parsingWhat == "name" {
			option += field + " "
		} else if parsingWhat == "value" {
			value += field + " "
		}
	}

	option = strings.TrimSuffix(option, " ")
	value = strings.TrimSuffix(value, " ")

	switch option {
	case "Hash":
		size, err := strconv.Atoi(value)
		if err == nil {
			inter.Search.TT.Unitialize()
			inter.Search.TT.Resize(uint64(size))
		}
	case "Clear Hash":
		inter.Search.TT.Clear()
	case "Clear History":
		inter.Search.ClearHistoryTable()
	case "UseBook":
		if value == "true" {
			inter.OptionUseBook = true
		} else if value == "false" {
			inter.OptionUseBook = false
		}
	case "BookPath":
		var err error
		inter.OpeningBook, err = LoadPolyglotFile(value)

		if err == nil {
			fmt.Println("Opening book loaded...")
		} else {
			fmt.Println("Failed to load opening book...")
		}
	case "BookMoveDelay":
		size, err := strconv.Atoi(value)
		if err == nil {
			inter.OptionBookMoveDelay = size
		}
	case "MiddleGameContempt":
		contempt, err := strconv.Atoi(value)
		if err == nil {
			MiddleGameDraw = int16(contempt)
		}
	case "EndGameContempt":
		contempt, err := strconv.Atoi(value)
		if err == nil {
			EndGameDraw = int16(contempt)
		}
	}
}

// Respond to the command "go"
func (inter *UCIInterface) goCommandResponse(command string) {
	if inter.OptionUseBook {
		if entries, ok := inter.OpeningBook[GenPolyglotHash(&inter.Search.Pos)]; ok {

			// To allow opening variety, randomly select a move from an entry matching
			// the current position.
			entry := entries[rand.Intn(len(entries))]
			move := MoveFromCoord(&inter.Search.Pos, entry.Move)

			if inter.Search.Pos.MoveIsPseduoLegal(move) {
				time.Sleep(time.Duration(inter.OptionBookMoveDelay) * time.Second)
				fmt.Printf("bestmove %v\n", move)
				return
			}
		}
	}

	command = strings.TrimPrefix(command, "go")
	command = strings.TrimPrefix(command, " ")
	fields := strings.Fields(command)

	colorPrefix := "b"
	if inter.Search.Pos.SideToMove == White {
		colorPrefix = "w"
	}

	// Parse the time left, increment, and moves to go from the command parameters.
	timeLeft, increment, movesToGo := int(InfiniteTime), int(NoValue), int(NoValue)
	specifiedDepth := uint64(MaxPly)
	specifiedNodes := uint64(math.MaxUint64)
	searchTime := uint64(NoValue)

	for index, field := range fields {
		if strings.HasPrefix(field, colorPrefix) {
			if strings.HasSuffix(field, "time") {
				timeLeft, _ = strconv.Atoi(fields[index+1])
			} else if strings.HasSuffix(field, "inc") {
				increment, _ = strconv.Atoi(fields[index+1])
			}
		} else if field == "movestogo" {
			movesToGo, _ = strconv.Atoi(fields[index+1])
		} else if field == "depth" {
			specifiedDepth, _ = strconv.ParseUint(fields[index+1], 10, 8)
		} else if field == "nodes" {
			specifiedNodes, _ = strconv.ParseUint(fields[index+1], 10, 64)
		} else if field == "movetime" {
			searchTime, _ = strconv.ParseUint(fields[index+1], 10, 64)
		}
	}

	// Set the timeLeft to NoValue if we're already given a move time
	// to use via movetime.
	if searchTime != uint64(NoValue) {
		timeLeft = int(NoValue)
	}

	// Setup the timer with the go command time control information.
	inter.Search.Timer.SetHardTimeForMove(int64(searchTime))
	inter.Search.Timer.TimeLeft = int64(timeLeft)
	inter.Search.Timer.Increment = int64(increment)
	inter.Search.Timer.MovesToGo = int64(movesToGo)

	// Setup user defined search options if given.
	inter.Search.SpecifiedDepth = uint8(specifiedDepth)
	inter.Search.SpecifiedNodes = specifiedNodes

	// Report the best move found by the engine to the GUI.
	bestMove := inter.Search.Search()
	fmt.Printf("bestmove %v\n", bestMove)
}

func (inter *UCIInterface) quitCommandResponse() {
	inter.Search.TT.Unitialize()
}

func (inter *UCIInterface) UCILoop() {
	rand.Seed(time.Now().Unix())
	reader := bufio.NewReader(os.Stdin)

	inter.uciCommandResponse()
	inter.Reset()

	inter.Search.TT.Resize(DefaultTTSize)
	inter.Search.Pos.LoadFEN(FENStartPosition)
	inter.OpeningBook = make(map[uint64][]PolyglotEntry)
	inter.OptionBookMoveDelay = DefaultBookMoveDelay

	for {
		command, _ := reader.ReadString('\n')
		command = strings.Replace(command, "\r\n", "\n", -1)

		if command == "uci\n" {
			inter.uciCommandResponse()
		} else if command == "isready\n" {
			fmt.Printf("readyok\n")
		} else if strings.HasPrefix(command, "setoption") {
			inter.setOptionCommandResponse(command)
		} else if strings.HasPrefix(command, "ucinewgame") {
			inter.Search.TT.Clear()
			inter.Search.ClearHistoryTable()
		} else if strings.HasPrefix(command, "position") {
			inter.positionCommandResponse(command)
		} else if strings.HasPrefix(command, "go") {
			go inter.goCommandResponse(command)
		} else if strings.HasPrefix(command, "stop") {
			inter.Search.Timer.Stop = true
		} else if command == "quit\n" {
			inter.quitCommandResponse()
			break
		}
	}
}
