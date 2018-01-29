package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/nsf/termbox-go"
	"github.com/pkg/errors"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"time"
	"unicode"
)

var debug bool

type state struct {
	currentLevel level
	status       string
	dict         []string
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

func newState() state {
	s := state{
		currentLevel: level{
			numWords:        3,
			activeWordIndex: -1,
			levelNumber:     0,
		},
		dict: readWords(),
	}
	s.newLevel()
	return s
}

func readWords() []string {
	ioutil.ReadFile("en.txt")
	file, err := os.Open("en.txt")
	if err != nil {
		panic(errors.Wrap(err, "couldn't open word file"))
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	words := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		words = append(words, line)
		if len(words) >= 1000000 {
			break
		}
	}
	return words
}

func (s *state) newLevel() {
	currentLevel := s.currentLevel
	s.currentLevel = level{
		numWords:        currentLevel.numWords + 1,
		activeWordIndex: -1,
		levelNumber:     currentLevel.levelNumber + 1,
	}
	wordSet := map[int]struct{}{}
	for len(s.currentLevel.words) < s.currentLevel.numWords {
		word := newWord(s.dict, wordSet)
		_, ok := wordSet[word.location.y]
		if !ok {
			wordSet[word.location.y] = struct{}{}
			s.currentLevel.words = append(s.currentLevel.words, word)
		}
	}
}

func newWord(dict []string, wordSet map[int]struct{}) word {
	_, height := termbox.Size()
	return word{
		text: dict[rand.Intn(len(dict))],
		location: point{
			x: 0,
			y: rand.Intn(height-1) + 1,
		},
		cursor: 0,
	}
}

func exit(events chan termbox.Event, timer <-chan time.Time) {
	close(events)
	termbox.Close()
}

func gameLoop(events chan termbox.Event, timer <-chan time.Time, gameState chan state) {
	s := newState()
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
								s.newLevel()
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

		if debug {
			drawDebugger(s)
		}

		drawText(0, 0, fmt.Sprintf("Level #%s", string(strconv.Itoa(s.currentLevel.levelNumber))))

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
	isDebug := flag.Bool("debug", false, "true or false")
	flag.Parse()
	debug = isDebug != nil && *isDebug

	rand.Seed(time.Now().Unix())
	readWords()
	err := termbox.Init()
	if err != nil {
		panic(errors.Wrap(err, "failed to init termbox"))
	}

	events := make(chan termbox.Event)
	timer := time.Tick(500 * time.Millisecond)
	gameState := make(chan state)

	go renderLoop(gameState)
	go eventLoop(events)
	defer exit(events, timer)

	gameLoop(events, timer, gameState)
}
