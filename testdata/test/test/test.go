package test

type Bar struct{}

func NewBar() (*Bar, error) {
	return &Bar{}, nil
}
