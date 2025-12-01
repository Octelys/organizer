package entities

type TableContent struct {
	Error   string              `json:"error"`
	Entries []TableContentEntry `json:"entries"`
}

type TableContentEntry struct {
	Title       string  `json:"title"`
	PageNumbers []uint8 `json:"pageNumbers"`
}
