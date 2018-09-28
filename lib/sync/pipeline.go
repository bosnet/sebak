package sync

func Pipeline(p Producer, c Consumer) error {
	if err := c.Consume(p.Produce()); err != nil {
		return err
	}
	if err := p.SetResponse(c.Response()); err != nil {
		return err
	}
	return nil
}
