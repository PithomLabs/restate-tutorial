package main

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/server"
	"github.com/starfederation/datastar-go/datastar"
	_ "modernc.org/sqlite"
)

//go:embed index.html
var indexHTML []byte

// Note: Create index.html file in the same directory as main.go

const (
	gridSize = 50
	cellSize = 10
)

type GameState struct {
	Grid       [][]bool `json:"grid"`
	Running    bool     `json:"running"`
	Generation int      `json:"generation"`
}

type GameService struct {
	db *sql.DB
}

// Initialize database
func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", "file:gameoflife.db?cache=shared")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS game_states (
			game_id TEXT PRIMARY KEY,
			state TEXT NOT NULL,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// Restate handler to get game state
func (g *GameService) GetState(ctx restate.ObjectSharedContext, _ restate.Void) (GameState, error) {
	gameID := restate.Key(ctx)

	// Try to get from Restate state first
	state, err := restate.Get[GameState](ctx, "game_state")
	if err != nil {
		return GameState{}, err
	}

	// If not in Restate state, check DB
	if state.Grid == nil {
		state = g.loadFromDB(gameID)
		if state.Grid == nil {
			// Initialize new game
			state = GameState{
				Grid:       makeEmptyGrid(),
				Running:    false,
				Generation: 0,
			}
		}
	}

	return state, nil
}

// Restate handler to update cell
func (g *GameService) ToggleCell(ctx restate.ObjectContext, req struct {
	X int `json:"x"`
	Y int `json:"y"`
}) (GameState, error) {
	state, err := restate.Get[GameState](ctx, "game_state")
	if err != nil || state.Grid == nil {
		state = GameState{
			Grid:       makeEmptyGrid(),
			Running:    false,
			Generation: 0,
		}
	}

	if req.X >= 0 && req.X < gridSize && req.Y >= 0 && req.Y < gridSize {
		state.Grid[req.Y][req.X] = !state.Grid[req.Y][req.X]
		restate.Set(ctx, "game_state", state)
		g.saveToDB(restate.Key(ctx), state)
	}

	return state, nil
}

// Restate handler to start/stop simulation
func (g *GameService) SetRunning(ctx restate.ObjectContext, running bool) (GameState, error) {
	state, err := restate.Get[GameState](ctx, "game_state")
	if err != nil || state.Grid == nil {
		state = GameState{
			Grid:       makeEmptyGrid(),
			Running:    false,
			Generation: 0,
		}
	}

	state.Running = running
	restate.Set(ctx, "game_state", state)
	g.saveToDB(restate.Key(ctx), state)

	return state, nil
}

// Restate handler to advance one generation
func (g *GameService) NextGeneration(ctx restate.ObjectContext, _ restate.Void) (GameState, error) {
	state, err := restate.Get[GameState](ctx, "game_state")
	if err != nil || state.Grid == nil {
		state = GameState{
			Grid:       makeEmptyGrid(),
			Running:    false,
			Generation: 0,
		}
	}

	state.Grid = computeNextGeneration(state.Grid)
	state.Generation++

	restate.Set(ctx, "game_state", state)
	g.saveToDB(restate.Key(ctx), state)

	return state, nil
}

// Restate handler to reset game
func (g *GameService) Reset(ctx restate.ObjectContext, _ restate.Void) (GameState, error) {
	state := GameState{
		Grid:       makeEmptyGrid(),
		Running:    false,
		Generation: 0,
	}

	restate.Set(ctx, "game_state", state)
	g.saveToDB(restate.Key(ctx), state)

	return state, nil
}

// Restate handler to load preset pattern
func (g *GameService) LoadPreset(ctx restate.ObjectContext, req struct {
	Pattern [][]int `json:"pattern"`
}) (GameState, error) {
	state := GameState{
		Grid:       makeEmptyGrid(),
		Running:    false,
		Generation: 0,
	}

	// Apply pattern
	offsetX := (gridSize - 15) / 2
	offsetY := (gridSize - 15) / 2

	for _, cell := range req.Pattern {
		if len(cell) == 2 {
			y := cell[0] + offsetY
			x := cell[1] + offsetX
			if x >= 0 && x < gridSize && y >= 0 && y < gridSize {
				state.Grid[y][x] = true
			}
		}
	}

	restate.Set(ctx, "game_state", state)
	g.saveToDB(restate.Key(ctx), state)

	return state, nil
}

// Database operations
func (g *GameService) saveToDB(gameID string, state GameState) {
	stateJSON, _ := json.Marshal(state)
	g.db.Exec(`
		INSERT OR REPLACE INTO game_states (game_id, state, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
	`, gameID, string(stateJSON))
}

func (g *GameService) loadFromDB(gameID string) GameState {
	var stateJSON string
	err := g.db.QueryRow(`
		SELECT state FROM game_states WHERE game_id = ?
	`, gameID).Scan(&stateJSON)

	if err != nil {
		return GameState{}
	}

	var state GameState
	json.Unmarshal([]byte(stateJSON), &state)
	return state
}

// Game of Life logic
func makeEmptyGrid() [][]bool {
	grid := make([][]bool, gridSize)
	for i := range grid {
		grid[i] = make([]bool, gridSize)
	}
	return grid
}

func countNeighbors(grid [][]bool, x, y int) int {
	count := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := x+dx, y+dy
			if nx >= 0 && nx < gridSize && ny >= 0 && ny < gridSize && grid[ny][nx] {
				count++
			}
		}
	}
	return count
}

func computeNextGeneration(grid [][]bool) [][]bool {
	newGrid := makeEmptyGrid()
	for y := 0; y < gridSize; y++ {
		for x := 0; x < gridSize; x++ {
			neighbors := countNeighbors(grid, x, y)
			if grid[y][x] {
				newGrid[y][x] = neighbors == 2 || neighbors == 3
			} else {
				newGrid[y][x] = neighbors == 3
			}
		}
	}
	return newGrid
}

func main() {
	db, err := initDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	gameService := &GameService{db: db}

	// Create Restate service
	restateService := restate.NewObject("GameOfLife").
		Handler("GetState", restate.NewObjectSharedHandler(gameService.GetState)).
		Handler("ToggleCell", restate.NewObjectHandler(gameService.ToggleCell)).
		Handler("SetRunning", restate.NewObjectHandler(gameService.SetRunning)).
		Handler("NextGeneration", restate.NewObjectHandler(gameService.NextGeneration)).
		Handler("Reset", restate.NewObjectHandler(gameService.Reset)).
		Handler("LoadPreset", restate.NewObjectHandler(gameService.LoadPreset))

		// Start Restate server

	go func() {
		if err := server.NewRestate().
			Bind(restateService).
			Start(context.Background(), ":6000"); err != nil {
			log.Fatal(err)
		}
	}()

	// HTTP server for frontend
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write(indexHTML)
	})

	// SSE endpoint for real-time updates
	r.Get("/game/{gameID}/stream", func(w http.ResponseWriter, r *http.Request) {
		gameID := chi.URLParam(r, "gameID")
		sse := datastar.NewSSE(w, r)

		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case <-ticker.C:
				// Get current state from DB
				state := gameService.loadFromDB(gameID)

				// Initialize if empty
				if state.Grid == nil {
					state = GameState{
						Grid:       makeEmptyGrid(),
						Running:    false,
						Generation: 0,
					}
					gameService.saveToDB(gameID, state)
				}

				if state.Running {
					// Advance generation
					state.Grid = computeNextGeneration(state.Grid)
					state.Generation++
					gameService.saveToDB(gameID, state)
				}

				// Send state to client
				if err := sse.MarshalAndPatchSignals(state); err != nil {
					return
				}
			}
		}
	})

	// API endpoints
	r.Post("/game/{gameID}/toggle", func(w http.ResponseWriter, r *http.Request) {
		gameID := chi.URLParam(r, "gameID")
		x, _ := strconv.Atoi(r.URL.Query().Get("x"))
		y, _ := strconv.Atoi(r.URL.Query().Get("y"))

		state := gameService.loadFromDB(gameID)
		if state.Grid == nil {
			state = GameState{Grid: makeEmptyGrid()}
		}

		if x >= 0 && x < gridSize && y >= 0 && y < gridSize {
			state.Grid[y][x] = !state.Grid[y][x]
			gameService.saveToDB(gameID, state)
		}

		json.NewEncoder(w).Encode(state)
	})

	r.Post("/game/{gameID}/running", func(w http.ResponseWriter, r *http.Request) {
		gameID := chi.URLParam(r, "gameID")
		running := r.URL.Query().Get("running") == "true"

		state := gameService.loadFromDB(gameID)
		if state.Grid == nil {
			state = GameState{Grid: makeEmptyGrid()}
		}

		state.Running = running
		gameService.saveToDB(gameID, state)

		json.NewEncoder(w).Encode(state)
	})

	r.Post("/game/{gameID}/reset", func(w http.ResponseWriter, r *http.Request) {
		gameID := chi.URLParam(r, "gameID")
		state := GameState{
			Grid:       makeEmptyGrid(),
			Running:    false,
			Generation: 0,
		}
		gameService.saveToDB(gameID, state)

		json.NewEncoder(w).Encode(state)
	})

	r.Post("/game/{gameID}/preset", func(w http.ResponseWriter, r *http.Request) {
		gameID := chi.URLParam(r, "gameID")

		var req struct {
			Pattern [][]int `json:"pattern"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		state := GameState{
			Grid:       makeEmptyGrid(),
			Running:    false,
			Generation: 0,
		}

		// Apply pattern
		offsetX := (gridSize - 15) / 2
		offsetY := (gridSize - 15) / 2

		for _, cell := range req.Pattern {
			if len(cell) == 2 {
				y := cell[0] + offsetY
				x := cell[1] + offsetX
				if x >= 0 && x < gridSize && y >= 0 && y < gridSize {
					state.Grid[y][x] = true
				}
			}
		}

		gameService.saveToDB(gameID, state)
		json.NewEncoder(w).Encode(state)
	})

	log.Println("Server starting on :8888")
	log.Println("Restate server on :6000")
	log.Fatal(http.ListenAndServe(":8888", r))
}
