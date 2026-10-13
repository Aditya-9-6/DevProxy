package waf

// SQLToken represents semantic tokens in SQL grammar.
type SQLToken int

const (
	TokenKeyword SQLToken = iota
	TokenOperator
	TokenLiteral
	TokenComment
	TokenTautology
)

// Transition defines a state machine transition for SQLi detection.
type Transition struct {
	From SQLToken
	To   SQLToken
	Risk bool
}

// Grammar defines the state machine rules.
var Grammar = []Transition{
	{TokenKeyword, TokenTautology, true},
	{TokenOperator, TokenTautology, true},
}
