package repository

type One struct {
}

func (r *One) One() {

}

func NewOne() *One {
	return &One{}
}

type Two struct {
}

func (r *Two) Two() {

}

func NewTwo() *Two {
	return &Two{}
}
