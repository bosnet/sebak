package sync

func Pipeline(p Producer, c Consumer) error {
	p.Produce(c.Response())
	return c.Consume(p.Message())
	// c.Consume(p.Produce())
	// p.SetResponse(c.Response())
	// or p.ProduceResponse(c.ConsumeResponse())
}
