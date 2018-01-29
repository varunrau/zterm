package main

import (
	"github.com/nsf/termbox-go"
	"github.com/pkg/errors"
	"strconv"
	"time"
	"unicode"
)

type state struct {
	currentLevel level
}

type point struct {
	x int
	y int
}

type level struct {
	numWords        int
	maxWordLength   int
	words           []word
	activeWordIndex int
	levelNumber     int
}

type word struct {
	text     string
	location point
	cursor   int
}

func (l *level) newLevel() level {
	return level{
		numWords:      l.numWords + 1,
		maxWordLength: l.maxWordLength,
		words: []word{
			{
				text: "hi",
				location: point{
					x: 10,
					y: 10,
				},
				cursor: 0,
			},
			{
				text: "hello",
				location: point{
					x: 30,
					y: 10,
				},
				cursor: 0,
			},
		},
		activeWordIndex: -1,
		levelNumber:     l.levelNumber + 1,
	}
}

func exit(events chan termbox.Event, timer <-chan time.Time) {
	close(events)
	termbox.Close()
}

func gameLoop(events chan termbox.Event, timer <-chan time.Time, gameState chan state) {
	s := state{
		currentLevel: level{
			numWords:      3,
			maxWordLength: 3,
			words: []word{
				{
					text: "wassup",
					location: point{
						x: 10,
						y: 10,
					},
					cursor: 0,
				},
				{
					text: "hello",
					location: point{
						x: 30,
						y: 10,
					},
					cursor: 0,
				},
			},
			activeWordIndex: -1,
			levelNumber:     1,
		},
	}

	// init game state
	gameState <- s

	for {
		select {
		case key := <-events:
			switch {
			case key.Key == termbox.KeyEsc || key.Key == termbox.KeyCtrlC: // exit
				return
			case unicode.IsLetter(key.Ch): // character
				if s.currentLevel.activeWordIndex == -1 {
					for i, word := range s.currentLevel.words {
						if word.text[0] == byte(key.Ch) {
							s.currentLevel.activeWordIndex = i
							s.currentLevel.words[i].cursor = 1
						}
					}
				} else {
					aIndex := s.currentLevel.activeWordIndex
					activeWord := s.currentLevel.words[aIndex]
					if activeWord.text[activeWord.cursor] == byte(key.Ch) {
						if len(activeWord.text) == activeWord.cursor+1 {
							// remove word
							s.currentLevel.words = append(s.currentLevel.words[:aIndex], s.currentLevel.words[aIndex+1:]...)
							s.currentLevel.activeWordIndex = -1
							if len(s.currentLevel.words) == 0 {
								s.currentLevel = s.currentLevel.newLevel()
							}
						} else {
							s.currentLevel.words[aIndex].cursor++
						}
					}
				}
			}
		case <-timer:
			for i := range s.currentLevel.words {
				s.currentLevel.words[i].location.x++
			}
		default:
			break
		}
		gameState <- s
	}
}

func renderLoop(gameState chan state) {
	for {
		s := <-gameState

		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
		drawDebugger(s)

		for i, word := range s.currentLevel.words {
			drawWord(word, i == s.currentLevel.activeWordIndex)
		}

		termbox.Flush()
	}
}

func drawWord(w word, active bool) {
	runes := []rune(w.text)
	for i, r := range runes {
		fgColor := termbox.ColorDefault
		if i == w.cursor && active {
			fgColor = termbox.ColorRed
		}
		termbox.SetCell(w.location.x+i, w.location.y, r, fgColor, termbox.ColorDefault)
	}
}

func drawDebugger(gameState state) {
	i := 0
	drawText(0, i, "DEBUG")
	i++
	for x, word := range gameState.currentLevel.words {
		if x == gameState.currentLevel.activeWordIndex {
			drawText(0, i+x, word.text+"*"+string(strconv.Itoa(word.cursor)))
		} else {
			drawText(0, i+x, word.text)
		}
	}
}

func drawText(x int, y int, str string) {
	runes := []rune(str)
	for i, r := range runes {
		termbox.SetCell(x+i, y, r, termbox.ColorRed, termbox.ColorDefault)
	}
}

func eventLoop(e chan termbox.Event) {
	for {
		e <- termbox.PollEvent()
	}
}

func main() {

	err := termbox.Init()
	if err != nil {
		panic(errors.Wrap(err, "failed to init termbox"))
	}

	events := make(chan termbox.Event)
	timer := time.Tick(1 * time.Second)
	gameState := make(chan state)

	go renderLoop(gameState)
	go eventLoop(events)
	defer exit(events, timer)

	gameLoop(events, timer, gameState)
}
