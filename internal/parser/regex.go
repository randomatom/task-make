package parser

import "regexp"

var (
	taskRegex        *regexp.Regexp
	importRegex      *regexp.Regexp
	interpreterRegex *regexp.Regexp
)

func init() {
	taskRegex = regexp.MustCompile(
		`^(\*?)(_?[A-Za-z][\w-]*)(?:/? */? *([A-Za-z][\w-]*))?:(?:\s+#!(.+))?$`,
	)

	importRegex = regexp.MustCompile(`^#!\s*import\s+([^\s]+)(?:\s+as\s+(\w+))?\s*$`)

	interpreterRegex = regexp.MustCompile(`^#!\s*(\w+)\s*$`)
}
