package debug

import (
	"regexp"
	"strconv"
)

var regRegex = regexp.MustCompile(
	`number="(\d+)",value="(0x[0-9a-fA-F]+)"`,
)

func ParseRegisters(
	lines []string,
) *Registers {

	regs := &Registers{}

	for _, line := range lines {

		matches := regRegex.FindAllStringSubmatch(
			line,
			-1,
		)

		for _, m := range matches {

			regNum, _ := strconv.Atoi(m[1])

			val, _ := strconv.ParseUint(
				m[2],
				0,
				32,
			)

			v := uint32(val)

			switch regNum {

			case 0:
				regs.R0 = v
			case 1:
				regs.R1 = v
			case 2:
				regs.R2 = v
			case 3:
				regs.R3 = v
			case 4:
				regs.R4 = v
			case 5:
				regs.R5 = v
			case 6:
				regs.R6 = v
			case 7:
				regs.R7 = v
			case 8:
				regs.R8 = v
			case 9:
				regs.R9 = v
			case 10:
				regs.R10 = v
			case 11:
				regs.R11 = v
			case 12:
				regs.R12 = v
			case 13:
				regs.SP = v
			case 14:
				regs.LR = v
			case 15:
				regs.PC = v
			case 25:
				regs.XPSR = v
			}
		}
	}

	return regs
}
