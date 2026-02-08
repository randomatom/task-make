package parser

import "regexp"

var (
	taskRegex        *regexp.Regexp
	taskHashBang     *regexp.Regexp
	reNumber         *regexp.Regexp
	importRegex      *regexp.Regexp
	interpreterRegex *regexp.Regexp
)

func init() {
	taskRegex = regexp.MustCompile(
		`^(\*?)(_?[A-Za-z][\w-]*)(?:/? */? *([A-Za-z][\w-]*))?:(?:\s+#!(.+))?$`,
	)

	taskHashBang = regexp.MustCompile(`:(?:\s+#!.+)$`)

	reNumber = regexp.MustCompile(`^\d+$`)

	importRegex = regexp.MustCompile(`^#!\s*import\s+([^\s]+)(?:\s+as\s+(\w+))?\s*$`)

	interpreterRegex = regexp.MustCompile(`^#!\s*(\w+)\s*$`)
}
