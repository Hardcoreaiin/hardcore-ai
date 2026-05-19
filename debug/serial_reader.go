package debug

import (
	"bufio"
	"strings"

	"go.bug.st/serial"
)

func StartSerialReader(
	portName string,
	baudRate int,
) {

	mode := &serial.Mode{
		BaudRate: baudRate,
	}

	port, err := serial.Open(
		portName,
		mode,
	)

	if err != nil {

		Log(
			ERROR,
			"SERIAL",
			err.Error(),
		)

		return
	}

	Log(INFO, "SERIAL", "Connected to "+portName)

	reader := bufio.NewReader(port)

	go func() {

		for {

			line, err := reader.ReadString('\n')

			if err != nil {

				Log(
					ERROR,
					"SERIAL",
					err.Error(),
				)

				continue
			}

			line = strings.TrimSpace(line)

			if line == "" {
				continue
			}

			ParseHardwareLog(line)

		}

	}()

}
