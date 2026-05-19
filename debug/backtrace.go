package debug

import (
	"regexp"
	"strconv"
)

type StackFrame struct {
	Level    int
	Function string
	File     string
	Fullname string
	Line     int
	Address  string
}

var frameRegex = regexp.MustCompile(
	`frame=\{level="(\d+)",addr="([^"]+)",func="([^"]+)",file="([^"]+)",fullname="([^"]+)",line="([^"]+)"`,
)

func ParseBacktrace(
	lines []string,
) []StackFrame {

	frames := []StackFrame{}

	for _, line := range lines {

		matches := frameRegex.FindAllStringSubmatch(
			line,
			-1,
		)

		for _, m := range matches {

			level, _ := strconv.Atoi(m[1])

			lineNum, _ := strconv.Atoi(m[6])

			frame := StackFrame{
				Level:    level,
				Address:  m[2],
				Function: m[3],
				File:     m[4],
				Fullname: m[5],
				Line:     lineNum,
			}

			frames = append(
				frames,
				frame,
			)
		}
	}

	return frames
}
