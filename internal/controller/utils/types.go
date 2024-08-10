package utils

type GithubReponseWantedProperties struct {
	Title       string `json:"title"`
	Description string `json:"body"`
	Number      int    `json:"number"`
	State       string `json:"state"`
}

type UpdatedValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type GithubPrResponseWantedProperties struct {
	Event string `json:"event"`
}
