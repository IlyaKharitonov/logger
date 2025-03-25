package logger

type LoggerConf struct {
	//уровни, которые напечатаются в консоль
	PrintInfo  bool `env:"LOGGER_PRINT_INFO"`
	PrintError bool `env:"LOGGER_PRINT_ERROR"`
	PrintDebug bool `env:"LOGGER_PRINT_DEBUG"`

	//уровни, которые запишутся в файлы
	WriteInfo  bool `env:"LOGGER_WRITE_INFO"`
	WriteError bool `env:"LOGGER_WRITE_ERROR"`
	WriteDebug bool `env:"LOGGER_WRITE_DEBUG"`

	BufferCapacity int    `env:"LOGGER_BUFFER_CAPACITY"` //размер буфера в который складываются логи пачкой из горутин перед записью в файл. похоже чем он меньше тем быстрее работает логгер
	ChanCapacity   int    `env:"LOGGER_CHAN_CAPACITY"`   //размер каналов в которые поступают сообщения
	ColorHeadings  bool   `env:"LOGGER_COLOR_HEADINGS"`  //раскрасить логи для лучшей визуализации
	DebugLog       bool   `env:"LOGGER_DEBUG_LOG"`       //дебаг логи самого логгера
	PathFolder     string `env:"LOGGER_PATH_FOLDER"`     //папка для сохранения
}

var loggerConfig *LoggerConf

func GetConfig() *LoggerConf {
	if loggerConfig == nil { //если данные есть в переменной, то возвращаем ее
		loggerConfig = &LoggerConf{}
	}
	return loggerConfig
}
