package app

// App is the composition root — dependencies are wired here.
type App struct{}

// New creates a new App instance.
func New() *App {
	return &App{}
}
