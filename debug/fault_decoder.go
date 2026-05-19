package debug

import "fmt"

type FaultReport struct {
	FaultType string

	Description string

	CFSR  uint32
	HFSR  uint32
	BFAR  uint32
	MMFAR uint32

	ProbableCause string
}

func DecodeFault(
	cfsr uint32,
	hfsr uint32,
	bfar uint32,
	mmfar uint32,
) *FaultReport {

	report := &FaultReport{
		CFSR:  cfsr,
		HFSR:  hfsr,
		BFAR:  bfar,
		MMFAR: mmfar,
	}

	// BUS FAULT

	if cfsr&(1<<1) != 0 {

		report.FaultType = "BUS FAULT"

		report.Description =
			"Precise data bus error"

		report.ProbableCause =
			"Invalid pointer or DMA corruption"

		return report
	}

	// MEMMANAGE

	if cfsr&(1<<0) != 0 {

		report.FaultType = "MEMMANAGE FAULT"

		report.Description =
			"Instruction access violation"

		report.ProbableCause =
			"Invalid execution region"

		return report
	}

	// USAGE FAULT

	if cfsr&(1<<16) != 0 {

		report.FaultType = "USAGE FAULT"

		report.Description =
			"Undefined instruction"

		report.ProbableCause =
			"Corrupted function pointer"

		return report
	}

	// HARDFAULT

	if hfsr&(1<<30) != 0 {

		report.FaultType = "HARD FAULT"

		report.Description =
			"Forced HardFault"

		report.ProbableCause =
			"Escalated configurable fault"

		return report
	}

	report.FaultType = "UNKNOWN"

	report.Description =
		"Unknown fault condition"

	report.ProbableCause =
		"Further analysis required"

	return report
}

func (f *FaultReport) Print() {

	fmt.Printf(
		"\nFAULT TYPE : %s\n",
		f.FaultType,
	)

	fmt.Printf(
		"DESCRIPTION : %s\n",
		f.Description,
	)

	fmt.Printf(
		"LIKELY CAUSE : %s\n",
		f.ProbableCause,
	)

	fmt.Printf(
		"\nCFSR  : 0x%08X\n",
		f.CFSR,
	)

	fmt.Printf(
		"HFSR  : 0x%08X\n",
		f.HFSR,
	)

	fmt.Printf(
		"BFAR  : 0x%08X\n",
		f.BFAR,
	)

	fmt.Printf(
		"MMFAR : 0x%08X\n",
		f.MMFAR,
	)
}
