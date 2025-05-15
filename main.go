package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

const (
	NumSystems         = 5
	MaxSystemValue     = 100
	MinSystemValue     = 0
	CriticalThreshold  = 20
	WarningThreshold   = 50
	StabilizeTime      = 5 * time.Second
	GameDuration       = 3 * time.Minute // 3 minutes to survive
	EventIntervalMin   = 8 * time.Second
	EventIntervalMax   = 15 * time.Second
	DegradationTick    = 750 * time.Millisecond
	InitialRepairKits  = 3
)

var systemNames = []string{"Coolant Flow", "Pressure Ctrl", "Core Temp", "Shield Integrity", "Power Output"}

// System struct
type System struct {
	ID              int
	Name            string
	Value           int
	DegradationRate int // How much it degrades per tick
	mu              sync.Mutex
	IsStable        bool // True if player action made it temporarily stable (during stabilization process)
}

func (s *System) Degrade() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.IsStable { // If being stabilized, degradation is paused for this system
		return
	}
	s.Value -= s.DegradationRate
	if s.Value < MinSystemValue {
		s.Value = MinSystemValue
	}
}

func (s *System) Boost(amount int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Value += amount
	if s.Value > MaxSystemValue {
		s.Value = MaxSystemValue
	}
}

func (s *System) Harm(amount int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Value -= amount
	if s.Value < MinSystemValue {
		s.Value = MinSystemValue
	}
}

// Game state
type Game struct {
	Systems       []*System
	EventLog      []string
	LogCapacity   int
	PlayerAction  string // e.g., "Stabilizing Core Temp..."
	ActionEndTime time.Time
	RepairKits    int
	GameOver      bool
	GameWon       bool
	StartTime     time.Time
	mu            sync.Mutex // For game-wide states like GameOver, EventLog, PlayerAction
}

func NewGame() *Game {
	g := &Game{
		Systems:     make([]*System, NumSystems),
		EventLog:    make([]string, 0, 10),
		LogCapacity: 10,
		RepairKits:  InitialRepairKits,
		StartTime:   time.Now(),
	}
	for i := 0; i < NumSystems; i++ {
		g.Systems[i] = &System{
			ID:              i,
			Name:            systemNames[i],
			Value:           MaxSystemValue - rand.Intn(20), // Start mostly stable
			DegradationRate: rand.Intn(3) + 2,             // Random degradation between 2-4
		}
	}
	return g
}

func (g *Game) AddLog(event string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	timestampedEvent := fmt.Sprintf("%s %s", time.Now().Format("15:04:05"), event)
	g.EventLog = append(g.EventLog, timestampedEvent)
	if len(g.EventLog) > g.LogCapacity {
		g.EventLog = g.EventLog[len(g.EventLog)-g.LogCapacity:] // Keep last N entries
	}
}

func (g *Game) SetPlayerAction(action string, duration time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.PlayerAction = action
	g.ActionEndTime = time.Now().Add(duration)
}

func (g *Game) ClearPlayerAction() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.PlayerAction = ""
}

func (g *Game) IsPlayerBusy() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.PlayerAction != "" && time.Now().Before(g.ActionEndTime)
}

// --- UI Functions ---
func clearScreen() {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		_ = cmd.Run() // Error ignored for simplicity
	} else {
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		_ = cmd.Run() // Error ignored for simplicity
	}
}

func (g *Game) Display() {
	clearScreen()
	fmt.Println(color.CyanString("--- REACTOR CONTROL TERMINAL ---"))
	g.mu.Lock() // Lock for game state relevant to display
	elapsed := time.Since(g.StartTime)
	kits := g.RepairKits
	playerAction := g.PlayerAction
	actionEndTime := g.ActionEndTime
	eventLogCopy := make([]string, len(g.EventLog))
	copy(eventLogCopy, g.EventLog)
	g.mu.Unlock()

	fmt.Printf("Time Elapsed: %s / %s\n", formatDuration(elapsed), formatDuration(GameDuration))
	fmt.Printf("Repair Kits: %d\n\n", kits)

	color.Yellow("SYSTEM STATUS:")
	for _, sys := range g.Systems {
		sys.mu.Lock()
		val := sys.Value
		name := sys.Name
		id := sys.ID
		sys.mu.Unlock()

		bar := renderBar(val, MaxSystemValue)
		var statusColorFormat string
		if val <= CriticalThreshold {
			statusColorFormat = color.New(color.FgRed, color.Bold).Sprintf("%3d/%3d", val, MaxSystemValue)
		} else if val <= WarningThreshold {
			statusColorFormat = color.New(color.FgYellow).Sprintf("%3d/%3d", val, MaxSystemValue)
		} else {
			statusColorFormat = color.New(color.FgGreen).Sprintf("%3d/%3d", val, MaxSystemValue)
		}
		fmt.Printf("[%d] %-18s: %s %s\n", id, name, statusColorFormat, bar)
	}

	if playerAction != "" {
		timeLeft := actionEndTime.Sub(time.Now())
		if timeLeft < 0 {
			timeLeft = 0
		}
		color.Magenta("\nCURRENT ACTION: %s (%.1fs left)", playerAction, timeLeft.Seconds())
	}

	fmt.Println(color.YellowString("\nEVENT LOG:"))
	for _, entry := range eventLogCopy { // Use the copied log
		lowerEntry := strings.ToLower(entry)
		if strings.Contains(lowerEntry, "critical") || strings.Contains(lowerEntry, "failed") || strings.Contains(lowerEntry, "catastrophic") {
			color.Red(entry)
		} else if strings.Contains(lowerEntry, "warning") || strings.Contains(lowerEntry, "event:") || strings.Contains(lowerEntry, "glitch") {
			color.Yellow(entry)
		} else if strings.Contains(lowerEntry, "success") || strings.Contains(lowerEntry, "complete") || strings.Contains(lowerEntry, "boost") {
			color.Green(entry)
		} else {
			fmt.Println(entry)
		}
	}

	fmt.Println(color.CyanString("\n--- AVAILABLE COMMANDS ---"))
	fmt.Println("  stabilize <id>          (Uses 1 Repair Kit, takes time)")
	fmt.Println("  divert <from_id> <to_id> <amount (10-30)>")
	fmt.Println("  vent <id>               (Risky, instant effect)")
	fmt.Println("  override <id>           (VERY Risky, instant effect)")
	fmt.Println("  quit")
	fmt.Print(color.CyanString("Enter command: "))
}

func renderBar(current, max int) string {
	barLength := 20
	fillLength := (current * barLength) / max
	if fillLength < 0 {
		fillLength = 0
	}
	if fillLength > barLength {
		fillLength = barLength
	}
	barStr := strings.Repeat("=", fillLength) + strings.Repeat("-", barLength-fillLength)

	if current <= CriticalThreshold {
		return color.RedString("[%s]", barStr)
	} else if current <= WarningThreshold {
		return color.YellowString("[%s]", barStr)
	}
	return color.GreenString("[%s]", barStr)
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02d:%02d", m, s)
}

// --- Game Logic Goroutines ---
func (g *Game) manageSystemDegradation(wg *sync.WaitGroup, quit <-chan struct{}) {
	defer wg.Done()
	ticker := time.NewTicker(DegradationTick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			g.mu.Lock()
			gameOver := g.GameOver
			gameWon := g.GameWon
			g.mu.Unlock()
			if gameOver || gameWon {
				return
			}
			for _, sys := range g.Systems {
				sys.Degrade() // Degrade handles its own lock
				sys.mu.Lock()
				val := sys.Value
				name := sys.Name
				id := sys.ID
				isStable := sys.IsStable
				sys.mu.Unlock()
				if val == MinSystemValue && !isStable {
					g.AddLog(color.RedString("CRITICAL: System %s (%d) at ZERO integrity!", name, id))
				}
			}
		case <-quit:
			return
		}
	}
}

func (g *Game) generateRandomEvents(wg *sync.WaitGroup, quit <-chan struct{}) {
	defer wg.Done()
	for {
		g.mu.Lock()
		gameOver := g.GameOver
		gameWon := g.GameWon
		g.mu.Unlock()
		if gameOver || gameWon {
			return // Exit if game has ended
		}

		sleepDuration := time.Duration(rand.Intn(int(EventIntervalMax-EventIntervalMin)) + int(EventIntervalMin))
		
		// Select with timeout for quit signal
		select {
		case <-time.After(sleepDuration):
			// Continue to trigger event
		case <-quit:
			return // Exit if quit signal received during sleep
		}

		g.mu.Lock()
		gameOver = g.GameOver // Re-check after sleep
		gameWon = g.GameWon
		g.mu.Unlock()
		if gameOver || gameWon {
			return
		}
		g.triggerRandomEvent()
	}
}


func (g *Game) triggerRandomEvent() {
	eventID := rand.Intn(5)
	sysID := rand.Intn(NumSystems)
	targetSystem := g.Systems[sysID]

	switch eventID {
	case 0:
		damage := rand.Intn(20) + 10
		targetSystem.Harm(damage)
		g.AddLog(color.YellowString("EVENT: Power surge in %s (%d)! Damage: %d", targetSystem.Name, sysID, damage))
	case 1:
		damage := rand.Intn(15) + 10
		targetSystem.Harm(damage)
		g.AddLog(color.YellowString("EVENT: Coolant leak detected near %s (%d)! Damage: %d", targetSystem.Name, sysID, damage))
		if targetSystem.ID == 0 && NumSystems > 2 && g.Systems[2].Name == "Core Temp" { // Assuming Coolant Flow is ID 0, Core Temp is ID 2
			coreTempSys := g.Systems[2]
			coreTempSys.mu.Lock()
			coreTempSys.DegradationRate += 1
			coreTempSys.mu.Unlock()
			g.AddLog(color.YellowString("INFO: Core Temp (%d) degradation increased due to coolant issue.", coreTempSys.ID))
		}
	case 2:
		g.AddLog(color.HiWhiteString("EVENT: Sensor glitch on %s (%d). Readings may be unreliable.", targetSystem.Name, sysID))
		targetSystem.mu.Lock()
		originalRate := targetSystem.DegradationRate
		targetSystem.DegradationRate += 2
		targetSystem.mu.Unlock()
		go func(sys *System, origRate int) {
			time.Sleep(15 * time.Second)
			sys.mu.Lock()
			sys.DegradationRate = origRate
			sys.mu.Unlock()
			g.AddLog(color.HiWhiteString("INFO: Sensor for %s (%d) recalibrated.", sys.Name, sys.ID))
		}(targetSystem, originalRate)
	case 3:
		boost := rand.Intn(10) + 5
		targetSystem.Boost(boost)
		g.AddLog(color.GreenString("EVENT: Unexpected efficiency boost in %s (%d)! Value +%d", targetSystem.Name, sysID, boost))
	case 4:
		numAffected := rand.Intn(NumSystems-1) + 1
		g.AddLog(color.YellowString("EVENT: Cosmic ray shower detected! Multiple systems affected."))
		affectedIndices := make(map[int]bool)
		for i := 0; i < numAffected; {
			idx := rand.Intn(NumSystems)
			if !affectedIndices[idx] {
				affectedIndices[idx] = true
				affectedSys := g.Systems[idx]
				damage := rand.Intn(5) + 5
				affectedSys.Harm(damage)
				g.AddLog(fmt.Sprintf("  - %s (%d) took %d damage.", affectedSys.Name, idx, damage))
				i++
			}
		}
	}
}

// --- Player Actions ---
func (g *Game) handleStabilize(sysID int) {
	if sysID < 0 || sysID >= NumSystems {
		g.AddLog(color.RedString("Error: Invalid system ID for stabilize."))
		return
	}
	if g.IsPlayerBusy() {
		g.AddLog(color.YellowString("Cannot start new action: Player busy."))
		return
	}
	g.mu.Lock()
	if g.RepairKits <= 0 {
		g.mu.Unlock()
		g.AddLog(color.RedString("Cannot stabilize: No repair kits left!"))
		return
	}
	g.RepairKits--
	g.mu.Unlock()

	targetSystem := g.Systems[sysID]
	g.SetPlayerAction(fmt.Sprintf("Stabilizing %s (%d)...", targetSystem.Name, sysID), StabilizeTime)
	g.AddLog(fmt.Sprintf("Commencing stabilization for %s (%d). This will take time.", targetSystem.Name, sysID))

	targetSystem.mu.Lock()
	targetSystem.IsStable = true
	targetSystem.mu.Unlock()

	go func(sys *System) {
		time.Sleep(StabilizeTime)

		sys.mu.Lock()
		sys.Value = MaxSystemValue
		sys.IsStable = false
		sys.mu.Unlock()

		g.ClearPlayerAction() // This goroutine is responsible for clearing its action
		g.AddLog(color.GreenString("System %s (%d) stabilization complete. Value restored to %d.", sys.Name, sys.ID, MaxSystemValue))
	}(targetSystem)
}

func (g *Game) handleDivert(fromSysID, toSysID, amount int) {
	if fromSysID < 0 || fromSysID >= NumSystems || toSysID < 0 || toSysID >= NumSystems || fromSysID == toSysID {
		g.AddLog(color.RedString("Error: Invalid system IDs for divert."))
		return
	}
	if amount < 10 || amount > 30 {
		g.AddLog(color.RedString("Error: Divert amount must be between 10 and 30."))
		return
	}
	if g.IsPlayerBusy() {
		g.AddLog(color.YellowString("Cannot divert: Player busy with another action."))
		return
	}

	fromSys := g.Systems[fromSysID]
	toSys := g.Systems[toSysID]

	fromSys.mu.Lock()
	canDivert := fromSys.Value >= amount+CriticalThreshold/2 // Less strict, can go into warning
	if !canDivert {
		fromSys.mu.Unlock()
		g.AddLog(color.RedString("Error: Not enough capacity in %s (%d) to divert %d.", fromSys.Name, fromSysID, amount))
		return
	}
	fromSys.Value -= amount
	fromSys.mu.Unlock()

	toSys.Boost(amount)
	g.AddLog(fmt.Sprintf("Diverted %d from %s (%d) to %s (%d).", amount, fromSys.Name, fromSysID, toSys.Name, toSysID))
}

func (g *Game) handleVent(sysID int) {
	if sysID < 0 || sysID >= NumSystems {
		g.AddLog(color.RedString("Error: Invalid system ID for vent."))
		return
	}
	if g.IsPlayerBusy() {
		g.AddLog(color.YellowString("Cannot vent: Player busy with another action."))
		return
	}

	targetSystem := g.Systems[sysID]
	targetSystem.mu.Lock()
	currentValue := targetSystem.Value
	targetSystem.mu.Unlock()

	boostAmount := (MaxSystemValue - currentValue) / 2
	if boostAmount < 10 {
		boostAmount = 10
	}
	if boostAmount == 0 && currentValue == MaxSystemValue { // No point venting if already max
	    g.AddLog(fmt.Sprintf("System %s (%d) is already optimal. Venting had no effect.", targetSystem.Name, sysID))
        return
    }
	targetSystem.Boost(boostAmount)
	g.AddLog(fmt.Sprintf("Emergency vent on %s (%d). Value increased by %d.", targetSystem.Name, sysID, boostAmount))

	if rand.Intn(100) < 35 {
		secondarySysID := rand.Intn(NumSystems)
		// Ensure secondary is not the same as vented, if possible and more than 1 system
		if NumSystems > 1 {
			for secondarySysID == sysID {
				secondarySysID = rand.Intn(NumSystems)
			}
		}
		secondaryDamage := rand.Intn(15) + 5
		g.Systems[secondarySysID].Harm(secondaryDamage)
		g.AddLog(color.RedString("WARNING: Vent caused backflow! System %s (%d) damaged by %d.", g.Systems[secondarySysID].Name, secondarySysID, secondaryDamage))
	}
}

func (g *Game) handleOverride(sysID int) {
	if sysID < 0 || sysID >= NumSystems {
		g.AddLog(color.RedString("Error: Invalid system ID for override."))
		return
	}
	if g.IsPlayerBusy() {
		g.AddLog(color.YellowString("Cannot override: Player busy with another action."))
		return
	}

	targetSystem := g.Systems[sysID]
	g.AddLog(color.HiRedString("Attempting DANGEROUS manual override on %s (%d)...", targetSystem.Name, sysID))
	time.Sleep(500 * time.Millisecond)

	outcome := rand.Intn(100)
	targetSystem.mu.Lock()
	name := targetSystem.Name // Store before potential nil dereference if game ends abruptly
	id := targetSystem.ID
	if outcome < 10 { // 10% success
		targetSystem.Value = MaxSystemValue
		g.AddLog(color.GreenString("OVERRIDE SUCCESS: %s (%d) fully stabilized!", name, id))
	} else if outcome < 40 { // 30% neutral
		g.AddLog(color.YellowString("OVERRIDE NEUTRAL: %s (%d) override had no significant effect.", name, id))
	} else { // 60% failure
		damage := rand.Intn(40) + 30
		targetSystem.Value -= damage
		if targetSystem.Value < MinSystemValue {
			targetSystem.Value = MinSystemValue
		}
		g.AddLog(color.RedString("OVERRIDE FAILED: %s (%d) CRITICAL DAMAGE! Value -%d", name, id, damage))
	}
	targetSystem.mu.Unlock()
}

// --- Main Game Loop ---
func main() {
	rand.Seed(time.Now().UnixNano())
	game := NewGame()
	quitSignal := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(1)
	go game.manageSystemDegradation(&wg, quitSignal)
	wg.Add(1)
	go game.generateRandomEvents(&wg, quitSignal)

	reader := bufio.NewReader(os.Stdin)
	uiTicker := time.NewTicker(200 * time.Millisecond) // UI refresh rate
	defer uiTicker.Stop()

	game.AddLog("SYSTEM BOOT: Reactor control online. Good luck, engineer.")

	inputChan := make(chan string)
	go func() { // Goroutine for blocking input read
		defer func() {
			// If ReadString panics (e.g. stdin closed abruptly), recover
			if r := recover(); r != nil {
				// Optionally log, but mainly prevent crash of this goroutine
			}
		}()
		for {
			rawInput, err := reader.ReadString('\n')
			if err != nil {
				// Likely EOF or other error, stop trying to read
				close(inputChan) // Signal main loop that input is done
				return
			}
			
			game.mu.Lock()
			isGameOverOrWon := game.GameOver || game.GameWon
			game.mu.Unlock()

			// Only send input if game is running, or if it's "quit" when game is over
			if !isGameOverOrWon || (isGameOverOrWon && strings.TrimSpace(strings.ToLower(rawInput)) == "quit") {
				select {
				case inputChan <- rawInput:
				case <-quitSignal: // If game is quitting, stop sending
					close(inputChan)
					return
				}
			}
		}
	}()

	running := true
	for running {
		game.Display()

		game.mu.Lock()
		isGameOver := game.GameOver
		isGameWon := game.GameWon
		game.mu.Unlock()

		if !isGameOver && !isGameWon {
			if time.Since(game.StartTime) >= GameDuration {
				game.mu.Lock()
				game.GameWon = true
				isGameWon = true // Update local var
				game.mu.Unlock()
				game.AddLog(color.HiGreenString("OBJECTIVE COMPLETE: Survived the critical period! You win!"))
			}

			criticalFailures := 0
			for _, sys := range game.Systems {
				sys.mu.Lock()
				val := sys.Value
				sys.mu.Unlock()
				if val <= MinSystemValue {
					criticalFailures++
				}
			}
			if criticalFailures >= 2 && !isGameOver { // Check against local isGameOver to prevent re-triggering
				game.mu.Lock()
				game.GameOver = true
				isGameOver = true // Update local var
				game.mu.Unlock()
				game.AddLog(color.HiRedString("CATASTROPHIC FAILURE: Multiple systems offline. Meltdown imminent. GAME OVER."))
			}
		}
		
		if isGameOver || isGameWon {
			game.Display() // One final display for win/loss message
			fmt.Println(color.CyanString("Game has ended. Type 'quit' or press Ctrl+C to exit."))
			// Wait for quit command via inputChan
		}

		var input string
		select {
		case <-uiTicker.C:
			// UI tick happened, just loop to Display again
			// Player action timeout is handled by the stabilize goroutine itself by calling ClearPlayerAction
			continue
		case rawInput, ok := <-inputChan:
			if !ok { // inputChan was closed
				running = false // End the game loop if input source is gone
				continue
			}
			input = strings.TrimSpace(rawInput)
		case <-quitSignal: // If the main quit signal is fired (e.g. future admin command)
		    running = false
			continue
		}

		parts := strings.Fields(strings.ToLower(input))
		if len(parts) == 0 {
			if isGameOver || isGameWon { // If game ended and user just presses Enter
				game.Display() // Keep displaying the end message
				fmt.Println(color.CyanString("Game has ended. Type 'quit' or press Ctrl+C to exit."))
			}
			continue
		}
		command := parts[0]

		if command == "quit" { // Allow quit anytime
			running = false
			game.AddLog("Exiting simulation...")
			continue
		}
		
		game.mu.Lock()
		isGameOver = game.GameOver // Re-check before processing non-quit command
		isGameWon = game.GameWon
		game.mu.Unlock()

		if isGameOver || isGameWon { // If game ended, only "quit" is processed above
			game.AddLog(color.WhiteString("Game ended. Only 'quit' is available."))
			continue
		}

		switch command {
		case "stabilize":
			if len(parts) < 2 {
				game.AddLog("Usage: stabilize <system_id>")
			} else if sysID, err := strconv.Atoi(parts[1]); err != nil {
				game.AddLog("Error: Invalid system ID format.")
			} else {
				game.handleStabilize(sysID)
			}
		case "divert":
			if len(parts) < 4 {
				game.AddLog("Usage: divert <from_id> <to_id> <amount>")
			} else {
				fromID, err1 := strconv.Atoi(parts[1])
				toID, err2 := strconv.Atoi(parts[2])
				amount, err3 := strconv.Atoi(parts[3])
				if err1 != nil || err2 != nil || err3 != nil {
					game.AddLog("Error: Invalid ID or amount format for divert.")
				} else {
					game.handleDivert(fromID, toID, amount)
				}
			}
		case "vent":
			if len(parts) < 2 {
				game.AddLog("Usage: vent <system_id>")
			} else if sysID, err := strconv.Atoi(parts[1]); err != nil {
				game.AddLog("Error: Invalid system ID format.")
			} else {
				game.handleVent(sysID)
			}
		case "override":
			if len(parts) < 2 {
				game.AddLog("Usage: override <system_id>")
			} else if sysID, err := strconv.Atoi(parts[1]); err != nil {
				game.AddLog("Error: Invalid system ID format.")
			} else {
				game.handleOverride(sysID)
			}
		default:
			game.AddLog(color.RedString("Unknown command: %s", command))
		}
	}

	close(quitSignal) // Signal all goroutines to stop
	// Input goroutine will also see quitSignal and close inputChan or exit.
	
	game.AddLog("Shutting down auxiliary systems...")
	game.Display() // Final display before exit
	fmt.Println(color.CyanString("Waiting for systems to power down..."))
	wg.Wait() // Wait for degradation and event goroutines
	fmt.Println(color.CyanString("All systems offline. Exiting."))
}
