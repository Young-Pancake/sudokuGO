# sudokuGo

sudokuGo is a terminal user interface sudoku game, written in Go.  
You can choose a difficulty and a seed or let the game pick a random seed to generate a puzzle.
The puzzles are fetched by my own [sudokuAPI](https://github.com/Young-Pancake/sudokuAPI)

Built with [Bubble Tea v2](https://charm.land/bubbletea) and [Lip Gloss v2](https://charm.land/lipgloss).

### If you don't have Go installed install it here:
### [Go install](https://go.dev/doc/install)

## Run

Clone the repo, then start it from the project directory:

```bash
go run .
```

## Build a binary

```bash
go build -o sudokuGo main.go
./sudokuGo
```

## Game Rules

1. Choose a difficulty.
2. Enter a seed, or use a random one.
3. Press `enter` to fetch the puzzle.
4. Use either arrow keys or wasd and enter the number to the empty blocks, backspace to remove cell.

Repeated digits in a row or a column will be highlighted in red. When the board matches the solution, the app shows solved!

## API

The game requests a puzzle from:

```
GET https://sudokuapi.fastapicloud.dev/sudokus/seed?seed=(someseed)&?difficulty=(either easy, medium, or hard)
```