package debug

type CPUStateStruct struct {
	PC    string
	SP    string
	LR    string
	XPSR  string
	State string
}

var CPUState CPUStateStruct

func UpdateRegister(
	current *string,
	newVal string,
) bool {

	if *current == newVal {
		return false
	}

	*current = newVal

	return true
}
