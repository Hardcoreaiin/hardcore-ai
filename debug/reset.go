package debug

func (g *GDBMI) Reset() error {

	err := g.Send(
		"monitor reset halt",
	)

	if err != nil {
		return err
	}

	_, err = g.Read()

	return err
}
