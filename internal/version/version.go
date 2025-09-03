package version

import _ "embed"

//go:embed COMMIT
var commit string

//go:embed VERSION
var number string

func Commit() string {
	return commit
}

func Number() string {
	return number
}
