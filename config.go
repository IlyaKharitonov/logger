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

	WriteTimout    uint   `env:"LOGGER_WRITE_TIMEOUT"`   //таймаут в секундах на запись из незаполненного буфера, если логов поступает немного и буфер не заполняется
	Format         string `env:"LOGGER_FORMAT"`          //формат записываемых в файл логов (строка или джейсон)
	BufferCapacity int    `env:"LOGGER_BUFFER_CAPACITY"` //размер буфера в который складываются логи пачкой из горутин перед записью в файл. лучшие результаты были при значении = 10-20
	ChanCapacity   int    `env:"LOGGER_CHAN_CAPACITY"`   //размер буфера каналов в которые поступают сообщения. сделал это чтобы не происходило блокировки горутины отправителя
	Color          bool   `env:"LOGGER_COLOR"`           //раскрасить уровень лога для лучшей визуализации в консоли
	DebugLog       bool   `env:"LOGGER_DEBUG_LOG"`       //дебаг логи самого логгера
	PathFolder     string `env:"LOGGER_PATH_FOLDER"`     //папка для сохранения логов
}

var loggerConfig *LoggerConf

func GetConfig() *LoggerConf {
	if loggerConfig == nil { //если данные есть в переменной, то возвращаем ее
		loggerConfig = &LoggerConf{}
	}
	return loggerConfig
}
