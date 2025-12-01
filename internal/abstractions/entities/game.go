package entities

type Game struct {
	Title   string `json:"title"`
	Console string `json:"console"`
	Score   int32  `json:"score"`
	OutOf   int32  `json:"outOf"`
}
