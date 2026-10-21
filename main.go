package main

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type screen int

const (
	screenSetup screen = iota
	screenLoading
	screenGame
	screenError
)

var difficulties = []string{"easy", "medium", "hard"}

type model struct {
	screen     screen
	width      int
	height     int
	diffIndex  int
	seedInput  string
	randomSeed bool
	difficulty string
	seed       int
	puzzle     [9][9]int
	solution   [9][9]int
	board      [9][9]int
	given      [9][9]bool
	row        int
	col        int
	won        bool
	err        string
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

func initialModel() model {
	return model{
		screen:     screenSetup,
		diffIndex:  1,
		randomSeed: true,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case puzzleMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			m.screen = screenError
			return m, nil
		}
		m.puzzle = msg.puzzle
		m.solution = msg.solution
		m.board = msg.puzzle
		m.given = givens(msg.puzzle)
		m.difficulty = msg.difficulty
		m.seed = msg.seed
		m.row = 0
		m.col = 0
		m.won = false
		m.err = ""
		m.screen = screenGame

	case tea.KeyPressMsg:
		key := msg.String()
		if key == "ctrl+c" {
			return m, tea.Quit
		}
		switch m.screen {
		case screenSetup:
			return m.updateSetup(key)
		case screenLoading:
			if key == "q" {
				return m, tea.Quit
			}
		case screenGame:
			return m.updateGame(key)
		case screenError:
			return m.updateError(key)
		}
	}
	return m, nil
}

func (m model) updateSetup(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q":
		return m, tea.Quit
	case "up", "w", "left", "a":
		if m.diffIndex > 0 {
			m.diffIndex--
		}
	case "down", "s", "right", "d":
		if m.diffIndex < len(difficulties)-1 {
			m.diffIndex++
		}
	case "r":
		m.randomSeed = !m.randomSeed
		m.err = ""
	case "backspace", "delete":
		if !m.randomSeed && len(m.seedInput) > 0 {
			m.seedInput = m.seedInput[:len(m.seedInput)-1]
		}
		m.err = ""
	case "enter":
		return m.startGame()
	default:
		if len(key) == 1 && key[0] >= '0' && key[0] <= '9' {
			if m.randomSeed {
				m.randomSeed = false
				m.seedInput = ""
			}
			if len(m.seedInput) < 9 {
				m.seedInput += key
			}
			m.err = ""
		}
	}
	return m, nil
}

func (m model) startGame() (tea.Model, tea.Cmd) {
	seed := 0
	if m.randomSeed {
		seed = rand.IntN(1_000_000_000) + 1
	} else {
		value, err := strconv.Atoi(m.seedInput)
		if err != nil {
			m.err = "type a seed or press r for a random one"
			return m, nil
		}
		seed = value
	}
	m.seed = seed
	m.difficulty = difficulties[m.diffIndex]
	m.err = ""
	m.screen = screenLoading
	return m, fetchPuzzle(m.difficulty, seed)
}

func (m model) updateGame(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q":
		return m, tea.Quit
	case "esc":
		m.screen = screenSetup
		m.won = false
		m.err = ""
	case "up", "w":
		if m.row > 0 {
			m.row--
		}
	case "down", "s":
		if m.row < 8 {
			m.row++
		}
	case "left", "a":
		if m.col > 0 {
			m.col--
		}
	case "right", "d":
		if m.col < 8 {
			m.col++
		}
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		if !m.given[m.row][m.col] && !m.won {
			m.board[m.row][m.col] = int(key[0] - '0')
			m.won = boardSolved(m.board, m.solution)
		}
	case "0", "backspace", "delete":
		if !m.given[m.row][m.col] && !m.won {
			m.board[m.row][m.col] = 0
		}
	}
	return m, nil
}

func (m model) updateError(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q":
		return m, tea.Quit
	case "enter", "r":
		m.err = ""
		m.screen = screenLoading
		return m, fetchPuzzle(m.difficulty, m.seed)
	case "esc":
		m.err = ""
		m.screen = screenSetup
	}
	return m, nil
}

func (m model) View() tea.View {
	view := tea.NewView(m.render())
	view.AltScreen = true
	view.WindowTitle = "SUdokuGo"
	return view
}

const apiURL = "https://sudokuapi.fastapicloud.dev/sudokus/seed"

const logo = `██████            ██        ██  ██        ██████  ████    
██     ██  ██     ██ █████  ██ ██  ██  ██ ██     ██  ██ 
█████  ██  ██ ██████ ██  ██ ████   ██  ██ ██ ███ ██  ██
    ██ ██  ██ ██  ██ ██  ██ ██ ██  ██  ██ ██  ██ ██  ██
██████ ██████ ██████ █████  ██  ██ ██████ ██████  ████ `

var (
	logoStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("5"))

	accentStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("6"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("9"))

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("5"))

	selectedConflictStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("0")).
				Background(lipgloss.Color("9"))

	givenStyle = lipgloss.NewStyle().
			Bold(true)

	userStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("6"))

	emptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	conflictStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("9"))

	winStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("2"))
)

type puzzleResponse struct {
	Puzzle       [][]int `json:"puzzle"`
	PuzzleSolved [][]int `json:"puzzle_solved"`
	Difficulty   string  `json:"difficulty"`
}

type puzzleMsg struct {
	puzzle     [9][9]int
	solution   [9][9]int
	difficulty string
	seed       int
	err        error
}

func fetchPuzzle(difficulty string, seed int) tea.Cmd {
	return func() tea.Msg {
		endpoint, err := url.Parse(apiURL)
		if err != nil {
			return puzzleMsg{err: err}
		}
		query := endpoint.Query()
		query.Set("seed", strconv.Itoa(seed))
		query.Set("difficulty", difficulty)
		endpoint.RawQuery = query.Encode()

		client := http.Client{Timeout: 10 * time.Second}
		response, err := client.Get(endpoint.String())
		if err != nil {
			return puzzleMsg{err: err}
		}
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			return puzzleMsg{err: fmt.Errorf("api returned %s", response.Status)}
		}

		var body puzzleResponse
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
			return puzzleMsg{err: err}
		}

		puzzle, ok := toGrid(body.Puzzle)
		if !ok {
			return puzzleMsg{err: fmt.Errorf("invalid puzzle in response")}
		}
		solution, ok := toGrid(body.PuzzleSolved)
		if !ok {
			return puzzleMsg{err: fmt.Errorf("invalid solution in response")}
		}
		if body.Difficulty == "" {
			body.Difficulty = difficulty
		}
		return puzzleMsg{
			puzzle:     puzzle,
			solution:   solution,
			difficulty: body.Difficulty,
			seed:       seed,
		}
	}
}

func toGrid(values [][]int) ([9][9]int, bool) {
	var grid [9][9]int
	if len(values) != 9 {
		return grid, false
	}
	for row, valuesRow := range values {
		if len(valuesRow) != 9 {
			return grid, false
		}
		for col, value := range valuesRow {
			if value < 0 || value > 9 {
				return grid, false
			}
			grid[row][col] = value
		}
	}
	return grid, true
}

func givens(puzzle [9][9]int) [9][9]bool {
	var given [9][9]bool
	for row := range puzzle {
		for col := range puzzle[row] {
			given[row][col] = puzzle[row][col] != 0
		}
	}
	return given
}

func boardSolved(board [9][9]int, solution [9][9]int) bool {
	return board == solution
}

func conflict(board [9][9]int, row int, col int) bool {
	value := board[row][col]
	if value == 0 {
		return false
	}
	for index := 0; index < 9; index++ {
		if index != col && board[row][index] == value {
			return true
		}
		if index != row && board[index][col] == value {
			return true
		}
	}
	boxRow := row / 3 * 3
	boxCol := col / 3 * 3
	for index := 0; index < 9; index++ {
		targetRow := boxRow + index/3
		targetCol := boxCol + index%3
		if (targetRow != row || targetCol != col) && board[targetRow][targetCol] == value {
			return true
		}
	}
	return false
}

func (m model) render() string {
	var content string
	switch m.screen {
	case screenSetup:
		content = m.renderSetup()
	case screenLoading:
		content = m.renderLoading()
	case screenGame:
		content = m.renderGame()
	case screenError:
		content = m.renderError()
	}
	width := m.width
	height := m.height
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}

func (m model) renderSetup() string {
	var content strings.Builder
	content.WriteString(logoStyle.Render(logo))
	content.WriteString("\n\n")
	content.WriteString(dimStyle.Render("difficulty"))
	content.WriteString("\n")
	for index, difficulty := range difficulties {
		if index == m.diffIndex {
			content.WriteString(selectedStyle.Render("▸ " + difficulty + "  "))
		} else {
			content.WriteString(dimStyle.Render("  " + difficulty + "  "))
		}
		content.WriteString("\n")
	}
	content.WriteString("\n")
	content.WriteString(dimStyle.Render("seed"))
	content.WriteString("\n")
	if m.randomSeed {
		content.WriteString(selectedStyle.Render(" random "))
		content.WriteString("  ")
		content.WriteString(dimStyle.Render("press r to enter a seed"))
	} else if m.seedInput == "" {
		content.WriteString(accentStyle.Render("▌"))
		content.WriteString(" ")
		content.WriteString(dimStyle.Render("type a number"))
	} else {
		content.WriteString(accentStyle.Render(m.seedInput + "▌"))
	}
	content.WriteString("\n\n")
	if m.err != "" {
		content.WriteString(errorStyle.Render(m.err))
		content.WriteString("\n\n")
	}
	content.WriteString(dimStyle.Render("↑/↓ or w/s  difficulty     digits  seed     r  random"))
	content.WriteString("\n")
	content.WriteString(dimStyle.Render("enter  play     q  quit"))
	return content.String()
}

func (m model) renderLoading() string {
	var content strings.Builder
	content.WriteString(logoStyle.Render("SUdokuGo"))
	content.WriteString("\n\n")
	content.WriteString(accentStyle.Render("fetching " + m.difficulty + " puzzle..."))
	content.WriteString("\n\n")
	content.WriteString(dimStyle.Render("seed " + strconv.Itoa(m.seed)))
	return content.String()
}

func (m model) renderError() string {
	var content strings.Builder
	content.WriteString(logoStyle.Render("SUdokuGo"))
	content.WriteString("\n\n")
	content.WriteString(errorStyle.Render("could not load a puzzle"))
	content.WriteString("\n\n")
	content.WriteString(dimStyle.Render(m.err))
	content.WriteString("\n\n")
	content.WriteString(dimStyle.Render("enter/r  retry     esc  back     q  quit"))
	return content.String()
}

func (m model) renderGame() string {
	var content strings.Builder
	content.WriteString(logoStyle.Render("SUdokuGo"))
	content.WriteString("  ")
	content.WriteString(dimStyle.Render(m.difficulty + " · seed " + strconv.Itoa(m.seed)))
	content.WriteString("\n\n")
	content.WriteString(m.renderBoard())
	content.WriteString("\n\n")
	if m.won {
		content.WriteString(winStyle.Render("✓ solved!"))
		content.WriteString("   ")
		content.WriteString(dimStyle.Render("esc  menu     q  quit"))
	} else {
		content.WriteString(dimStyle.Render("wasd / arrows  move     1-9  fill     0  clear"))
		content.WriteString("\n")
		content.WriteString(dimStyle.Render("esc  menu     q  quit"))
	}
	return content.String()
}

func (m model) renderBoard() string {
	var content strings.Builder
	content.WriteString("┌───────────┬───────────┬───────────┐\n")
	for row := 0; row < 9; row++ {
		content.WriteString("│")
		for col := 0; col < 9; col++ {
			content.WriteString(m.renderCell(row, col))
			if col == 8 {
				break
			}
			if col%3 == 2 {
				content.WriteString("│")
			} else {
				content.WriteString(" ")
			}
		}
		content.WriteString("│\n")
		if row < 8 && (row+1)%3 == 0 {
			content.WriteString("├───────────┼───────────┼───────────┤\n")
		}
	}
	content.WriteString("└───────────┴───────────┴───────────┘")
	return content.String()
}

func (m model) renderCell(row int, col int) string {
	value := m.board[row][col]
	text := "."
	if value != 0 {
		text = strconv.Itoa(value)
	}
	selected := row == m.row && col == m.col
	bad := value != 0 && conflict(m.board, row, col)
	style := emptyStyle
	switch {
	case selected && bad:
		style = selectedConflictStyle
	case selected:
		style = selectedStyle
	case bad:
		style = conflictStyle
	case m.given[row][col]:
		style = givenStyle
	case value != 0:
		style = userStyle
	}
	return style.Width(3).Align(lipgloss.Center).Render(text)
}
