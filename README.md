# Reactor Core Meltdown

Welcome to **Reactor Core Meltdown**! A highly challenging, text-based terminal game written in Go. Your mission, should you choose to accept it, is to prevent a catastrophic reactor meltdown by managing multiple, interconnected, and asynchronously failing systems under extreme pressure.

This game is designed as an exercise in Go programming, with a strong focus on goroutines, channels, and managing concurrent operations to create a difficult and fast-paced experience.

## Features

*   **Asynchronous System Degradation:** Each of the 5 reactor systems degrades independently and concurrently.
*   **Dynamic Random Events:** Unpredictable events (power surges, coolant leaks, sensor glitches, cosmic rays) will strike, further complicating your efforts.
*   **Time-Sensitive Player Actions:** Actions like `stabilize` take time, during which other systems continue to deteriorate.
*   **Resource Management:** You have a limited number of repair kits for stabilization.
*   **High-Stakes Decisions:** Risky commands like `vent` and `override` offer potential salvation or accelerated doom.
*   **Colorful Terminal UI:** A clear, color-coded dashboard provides real-time status of all systems and an event log.
*   **Intense Difficulty:** Designed to be very challenging, requiring quick thinking, prioritization, and a bit of luck!

## Requirements

*   Go (version 1.22 or newer recommended, as per your `go.mod`)
*   A terminal that supports ANSI escape codes for colors (most modern terminals do).

## Setup & Installation

1.  **Clone the repository (or download the files):**
    ```bash
    # If you have git installed and this becomes a git repository:
    # git clone <repository-url>
    # cd reactor_meltdown
    ```
    If you just have the `main.go`, `go.mod`, and `go.sum` files, ensure they are in a dedicated directory (e.g., `reactor_meltdown`).

2.  **Navigate to the project directory:**
    ```bash
    cd path/to/reactor_meltdown
    ```

3.  **Fetch dependencies:**
    This ensures the `github.com/fatih/color` package is available.
    ```bash
    go mod tidy
    ```

## Running the Game

There are two main ways to run the game:

1.  **Build and Run:**
    *   Build the executable:
        ```bash
        go build main.go
        ```
        (This will create `main.exe` on Windows or `main` on Linux/macOS in the current directory)
    *   Run the game:
        *   On Windows: `.\main.exe`
        *   On Linux/macOS: `./main`

2.  **Run Directly (without explicit build step):**
    ```bash
    go run main.go
    ```

## How to Play

*   **Objective:** Survive for the designated time (currently 3 minutes) without letting two or more reactor systems reach critical failure (0 integrity).
*   **The Terminal Interface:**
    *   **System Status:** Displays the current integrity percentage and a visual bar for each of the 5 reactor systems.
        *   <span style="color:green;">Green</span>: System stable.
        *   <span style="color:yellow;">Yellow</span>: System in warning state.
        *   <span style="color:red;">Red</span>: System in critical condition!
    *   **Event Log:** Shows incoming random events, outcomes of your actions, and critical warnings.
    *   **Player Action:** Indicates if you are currently busy with a timed action (e.g., "Stabilizing Core Temp...").
    *   **Repair Kits:** Shows how many repair kits you have left.
*   **Available Commands:**
    *   `stabilize <system_id>`:
        *   Initiates a stabilization process on the specified system (ID 0-4).
        *   Consumes 1 Repair Kit.
        *   Takes time (`StabilizeTime`, currently 5 seconds), during which you cannot perform other major actions.
        *   If successful, restores the system to 100% integrity.
    *   `divert <from_id> <to_id> <amount>`:
        *   Transfers a specified `amount` (10-30) of integrity from one system to another.
        *   Instantaneous, but depletes the source system.
        *   Cannot divert if the source system would drop too low.
    *   `vent <system_id>`:
        *   Performs an emergency vent on the specified system.
        *   Instantly boosts the system's integrity (typically by half the missing amount).
        *   Risky: Has a chance (35%) of causing secondary damage to another random system.
    *   `override <id>`:
        *   A **VERY** risky last-ditch effort to fix a system.
        *   Outcomes:
            *   Small chance (10%) of full stabilization.
            *   Moderate chance (30%) of no effect.
            *   High chance (60%) of causing significant critical damage to the system.
    *   `quit`: Exits the game.

*   **Tips for Survival:**
    *   Keep a close eye on all systems simultaneously.
    *   Prioritize which system to `stabilize` as you only have limited repair kits and can only stabilize one at a time.
    *   Use `divert` strategically for quick boosts, but be mindful of the source system.
    *   `vent` and `override` are desperate measures. Use them wisely!
    *   React quickly to random events; they can quickly escalate problems.

## Technology Stack

*   **Language:** Golang
*   **Concurrency:** Goroutines and Channels for asynchronous system degradation, event generation, and timed player actions.
*   **Synchronization:** `sync.Mutex` for ensuring safe concurrent access to shared game and system states.
*   **Terminal UI:** `github.com/fatih/color` for colored text output.

Good luck, Engineer. The fate of the reactor is in your hands!
