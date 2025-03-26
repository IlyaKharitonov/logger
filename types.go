package logger

type recordType struct {
	TimeUTC int64    `json:"timeUTC"` //нужно для сортировки
	Date    string   `json:"date"`
	Level   string   `json:"level"`
	Message string   `json:"message"`
	Params  []string `json:"params, omitempty"`
	Error   *string  `json:"error, omitempty"`
}
