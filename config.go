package logger

type LoggerConf struct {
	PathFolder       string   `yaml:"pathFolder"`
	PrintableLevels  []string `yaml:"printableLevels"`
	RecordableLevels []string `yaml:"recordableLevels"`
	Limit            int      `yaml:"limit"` //размер буфера в который складываются логи пачкой из горутин перед записью в файл. похоже чем он меньше тем быстрее работает логгер
	ColorHeadings    bool     `yaml:"colorHeadings"`
	DebugLog         bool     `yaml:"debugLog"`
	TerminalLogs     bool     `yaml:"terminalLogs"`
	FileLogs         bool     `yaml:"fileLogs"`
}

var loggerConfig *LoggerConf

func GetConfig() *LoggerConf {
	if loggerConfig == nil { //если данные есть в переменной, то возвращаем ее
		loggerConfig = &LoggerConf{}
	}
	return loggerConfig
}
