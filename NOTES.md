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