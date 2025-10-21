# File Transfer TUI

## Go Fundamentals

### Structs
- like objects in other languages, but no inheritance
- fields are the data, methhods are defined separately
- capitalized names are exported (public), lowercase are private

### Interfaces
- implicit implementation
- if a type has the right methods, it satisfies the interface
- the empty interface `interface{}` can hold any type
- https://gobyexample.com/interfaces

### Pointers
- `*Type` is a pointer to Type
- `&variable` gets the address of a variable
- methods can have pointer receivers `(m *model)` or value receivers `(m model)`
- pointer receivers let you  mutate the original, value receivers work on a copy

### Channels
- typed pipes for communication between goroutines
- connect concurrent goroutines
- can send values into channels from one goroutine and receive those values into another goroutine
- `ch := make(chan int)` creates an integer channel
- `ch <- value` sends a value to a channel
- `value := <-ch` receives a value from a channel

### Goroutines
- lightweight threads managed by the Go runtime
- `go functionName()` starts a function in a new goroutine
- incredibly cheap, can spawn thousands


---

## The Elm Architecture (TEA)

The pattern that Bubbletea uses. Everything flows in one direction:
    user input -> message -> update (changes state) -> view (renders state) -> display

### Key Concepts:

**Model**
- holds all of the application state
- one single source of truth
- immutable - return a new model on each update

**Messages**
- everything that happens is a message
- key press? message. timer tick? message. file transfer done? message.
0 messages are just Go types, usually structs

**Update Function**
- receives the current model and a message
- returns a new model and optionally a command
- this is where all state changes happen

**View Function**
- takes the model and returns a string to display
- pure function - same model always produces the same output
- no side effects

**Commands**
- functions that do I/O and eventually produce a message
- network requests, file operations, timers, etc.
- bubbletea runs these and sends their result back as a message

---

## Bubbletea Patterns

### Basic Loop

func (m model) Init tea.Cmd {
    // runs once the app starts
    // return any commands to run immediately
    return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // handles all messages
    // returns updated model and optional command
    switch msg := msg.(type) {
        case tea.KeyMsg:
        // handle keyboard input
    }
    return m, nil
}

func (m model) View() string {
    // renders the tui
    // just returns a string
    return "hello world"
}

### Key Concepts:

**Model**
- A single struct holding all of your application's state. The single source of truth.

**Messages (`tea.Msg`)**
- Structs that represent events (e.g., a key press, a timer tick, data received).
- Messages are the *only* way state can change.

**Update Function (`Update(tea.Msg)`)**
- A pure function that takes the current model and a message.
- It returns the *new* model state and an optional command.
- **Crucially, it has no side effects.** All I/O is handled by commands.

**View Function (`View()`)**
- A pure function that takes the model and returns a `string` to be rendered to the terminal.
- It's a direct representation of the current state.

**Commands (`tea.Cmd`)**
- Functions that perform I/O or other side effects (e.g., network requests, file operations, timers).
- They run in the background and return a `tea.Msg` when complete.

---

## This Project's Architecture

### State Machine
1.  **Idle**: User browses files with `filepicker`, enters destination in `textinput`.
2.  **Transferring**: A `transferring` boolean is set to `true`. The UI is locked, and progress is displayed.
3.  **Complete**: The `transferring` boolean is set to `false`. A final status message is shown, and the state is reset for the next transfer.

### Data Flow for a Transfer
1.  User hits `enter` in the destination input.
2.  `Update()` checks conditions (files selected, destination not empty).
3.  `Update()` returns a `tea.Cmd` that calls our `startTransfer()` function.
4.  `startTransfer()` runs in a background goroutine. It connects via SSH and starts another goroutine for the actual file transfers.
5.  It immediately returns a `transferStartedMsg` containing a `progressChan` back to the `Update()` function.
6.  The `Update()` function receives this message and stores the `progressChan` in the model.
7.  A `TickMsg` fires every 100ms, causing `Update()` to check the `progressChan` for new `TransferProgressMessage`s.
8.  As messages arrive, the `fileProgress` map and overall `progress` bar are updated, triggering a re-render by `View()`.
9.  When the transfer goroutine finishes, it `close()`s the `progressChan`.
10. The `TickMsg` handler sees the channel is closed and sets the final "Complete" status.

---

## Key Learnings from This Project

#### 1. Managing Async I/O in Bubbletea
- You cannot block the `Update` function. All long-running tasks must be in a `tea.Cmd`.
- The most robust pattern for progress updates is to have the command return a message containing a channel. Then, use a periodic `tea.Tick` message to poll that channel for new updates without blocking.

#### 2. Go Dependency Management
- `go.mod` tracks direct dependencies, while `go.sum` tracks the checksums of all direct and indirect dependencies.
- `go mod tidy` is the essential command to synchronize your `go.mod`/`go.sum` files with your code's `import` statements.
- Pinning to a specific version (`go get github.com/charmbracelet/bubbles@v0.17.0`) is a critical tool for ensuring stable builds when library APIs change.

#### 3. Implementing the SCP Protocol
- SCP is not a built-in function; it's a protocol running over an SSH session's standard I/O.
- The client-side implementation requires sending specifically formatted header strings to the remote `scp -t` (sink mode) command, like `C0644 [size] [filename]\n`, before sending the file's binary content.

---

## Future Improvements

- [ ] Resume incomplete transfers.
- [ ] Show transfer speed (bytes/sec).
- [ ] Support password-based SSH auth.
- [ ] Recursive directory transfer support.
- [ ] Proper host key verification instead of `InsecureIgnoreHostKey()`.
- [ ] Support for using an SSH agent.
- [ ] A config file to save favorite destinations.

---

## Resources

- [Bubbletea Docs](https://github.com/charmbracelet/bubbletea)
- [Bubbles Components](https://github.com/charmbracelet/bubbles)
- [Lipgloss Styling](https://github.com/charmbracelet/lipgloss)
- [Go by Example](https://gobyexample.com/)### Key Concepts:

**Model**
- A single struct holding all of your application's state. The single source of truth.

**Messages (`tea.Msg`)**
- Structs that represent events (e.g., a key press, a timer tick, data received).
- Messages are the *only* way state can change.

**Update Function (`Update(tea.Msg)`)**
- A pure function that takes the current model and a message.
- It returns the *new* model state and an optional command.
- **Crucially, it has no side effects.** All I/O is handled by commands.

**View Function (`View()`)**
- A pure function that takes the model and returns a `string` to be rendered to the terminal.
- It's a direct representation of the current state.

**Commands (`tea.Cmd`)**
- Functions that perform I/O or other side effects (e.g., network requests, file operations, timers).
- They run in the background and return a `tea.Msg` when complete.

---

## This Project's Architecture

### State Machine
1.  **Idle**: User browses files with `filepicker`, enters destination in `textinput`.
2.  **Transferring**: A `transferring` boolean is set to `true`. The UI is locked, and progress is displayed.
3.  **Complete**: The `transferring` boolean is set to `false`. A final status message is shown, and the state is reset for the next transfer.

### Data Flow for a Transfer
1.  User hits `enter` in the destination input.
2.  `Update()` checks conditions (files selected, destination not empty).
3.  `Update()` returns a `tea.Cmd` that calls our `startTransfer()` function.
4.  `startTransfer()` runs in a background goroutine. It connects via SSH and starts another goroutine for the actual file transfers.
5.  It immediately returns a `transferStartedMsg` containing a `progressChan` back to the `Update()` function.
6.  The `Update()` function receives this message and stores the `progressChan` in the model.
7.  A `TickMsg` fires every 100ms, causing `Update()` to check the `progressChan` for new `TransferProgressMessage`s.
8.  As messages arrive, the `fileProgress` map and overall `progress` bar are updated, triggering a re-render by `View()`.
9.  When the transfer goroutine finishes, it `close()`s the `progressChan`.
10. The `TickMsg` handler sees the channel is closed and sets the final "Complete" status.

---

## Key Learnings from This Project

#### 1. Managing Async I/O in Bubbletea
- You cannot block the `Update` function. All long-running tasks must be in a `tea.Cmd`.
- The most robust pattern for progress updates is to have the command return a message containing a channel. Then, use a periodic `tea.Tick` message to poll that channel for new updates without blocking.

#### 2. Go Dependency Management
- `go.mod` tracks direct dependencies, while `go.sum` tracks the checksums of all direct and indirect dependencies.
- `go mod tidy` is the essential command to synchronize your `go.mod`/`go.sum` files with your code's `import` statements.
- Pinning to a specific version (`go get github.com/charmbracelet/bubbles@v0.17.0`) is a critical tool for ensuring stable builds when library APIs change.

#### 3. Implementing the SCP Protocol
- SCP is not a built-in function; it's a protocol running over an SSH session's standard I/O.
- The client-side implementation requires sending specifically formatted header strings to the remote `scp -t` (sink mode) command, like `C0644 [size] [filename]\n`, before sending the file's binary content.

---

## Future Improvements

- [ ] Resume incomplete transfers.
- [ ] Show transfer speed (bytes/sec).
- [ ] Support password-based SSH auth.
- [ ] Recursive directory transfer support.
- [ ] Proper host key verification instead of `InsecureIgnoreHostKey()`.
- [ ] Support for using an SSH agent.
- [ ] A config file to save favorite destinations.

---

## Resources

- [Bubbletea Docs](https://github.com/charmbracelet/bubbletea)
- [Bubbles Components](https://github.com/charmbracelet/bubbles)
- [Lipgloss Styling](https://github.com/charmbracelet/lipgloss)
- [Go by Example](https://gobyexample.com/)