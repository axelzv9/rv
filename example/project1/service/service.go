package service

func NewOne(one repoOne, two repoTwo) *One {
	return &One{}
}

func NewTwo(two repoTwo) *Two {
	return &Two{}
}

type repoOne interface {
	One()
}

type repoTwo interface {
	Two()
}

type One struct {
}

func (s *One) MethodOne() {

}

type Two struct {
}

func (s *Two) MethodTwo() {

}
