package tgo

type Command struct {
	Command     string
	HelpMessage func(*Context) string
	Handler     Handler
}

type commandGroup struct {
	name     func(*Context) string
	commands []Command
}
