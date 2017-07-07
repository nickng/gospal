package migoinfer

import "go.uber.org/zap"

// Logger encapsulates a Logger and module which it belongs to.
// Use this through SetLogger() of visitor.
type Logger struct {
	*zap.SugaredLogger
	module string
}

type LogSetter interface {
	SetLogger(*Logger)
}

// Module returns (stylised) module name.
func (l *Logger) Module() string {
	return l.module
}
