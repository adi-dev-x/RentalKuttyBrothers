package model

type GenericUpdate struct {
	Code    string `json:"code"`
	Table   string `json:"table"`
	Key     string `json:"key"`
	Field   string `json:"field"`
	Status  string `json:"status"`
	ID      string `json:"id"`
	IDvalue string `json:"idvalue"`
}

type Fields struct {
	Table string `json:"table"`
	Key   string `json:"key"`
	ID    string `json:"id"`
}
