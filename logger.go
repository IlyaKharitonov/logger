package logger

import (
	"context"
	"fmt"
	"sync"
)

type ILogger interface {
	Info(msg string)  //Информационные сообщения о ходе работы программы
	Debug(msg string) //Сообщения отладки
	//возможно доработать, чтобы он печатал запрос в удобной форме и с аргументами
	Query(msg string) //Запросы в базу и тд. Возможно писать их в отдельный файл полезно
	//Critical(msg string) //Что-то очень важное, например паника, ошибка внешнего сервиса. Может тоже убрать и все в уровне Error обрабатывать
	//Warning(msg string)  //Предупреждение хз о чем. Возможно его убрать
	Error(msg string) //Ошибка в ходе работы программы
	//Fatal(msg string)

	Stop()
}

type logger struct {
	infoChan  chan msgType
	debugChan chan msgType
	queryChan chan msgType
	//criticalChan chan msgType
	//warningChan  chan msgType
	errorChan chan msgType

	limit            int //предел заполненности слайса, после которого логги из него будут считаны и отправлены на сортировку
	pathFolder       string
	printableLevels  []string //Уровни, которые нужно распечатать в терминал
	recordableLevels []string //Уровни, которые нужно записать в файл
	colorHeadings    bool     //вкл/выкл покраску заголовков в файлах логов. покрашенные заголовки упрощают чтение лога из консоли
	terminalLogs     bool
	fileLogs         bool

	stop    bool
	stopped chan struct{}
	wg      *sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc

	debugLog bool
}

func New(config *LoggerConf) *logger {
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	logger := &logger{
		infoChan:  make(chan msgType, 1000),
		debugChan: make(chan msgType, 1000),
		queryChan: make(chan msgType, 1000),
		//criticalChan: make(chan msgType, 1000),
		//warningChan:  make(chan msgType, 1000),
		errorChan: make(chan msgType, 1000),

		pathFolder:       config.PathFolder,
		printableLevels:  config.PrintableLevels,
		recordableLevels: config.RecordableLevels,
		limit:            config.Limit,
		colorHeadings:    config.ColorHeadings,
		terminalLogs:     config.TerminalLogs,
		fileLogs:         config.FileLogs,

		wg:       wg,
		stopped:  make(chan struct{}),
		ctx:      ctx,
		cancel:   cancel,
		debugLog: config.DebugLog,
	}

	if logger.fileLogs == true {
		go logger.StartProcessingLogs()
	}

	return logger
}

// читает из каналов логи и пишет их в слайсы, для дальнейшей обработки и записи
func (l *logger) StartProcessingLogs() {
	/*пуск горутин на каждый уровень логирования, указанный в конфигурации*/
	for _, rl := range l.recordableLevels {
		l.debug(fmt.Sprintf("пуск горутины для канала %s", rl))

		l.wg.Add(1)
		go l.listenChan(rl)
	}

	l.debug("жду в вызывающей горутине")

	l.wg.Wait()

	l.debug("отправляю сигнал о выполненной остановке в вызывающей горутине")

	l.stopped <- struct{}{}
}

// Stop() graceful stop
func (l *logger) Stop() {
	if l.fileLogs == false {
		return
	}
	l.debug("отправляю сигнал на остановку")
	l.cancel()
	l.debug("жду завершения работы горутин")
	<-l.stopped
	l.debug("логгер завершил работу")

}

func (l *logger) Info(msg string) {
	if l.terminalLogs == true && l.needPrint(Info) == true {
		fmt.Println(makeMessageColorful(Info, msg))
	}

	if l.fileLogs == true && l.needWrite(Info) == true {
		l.infoChan <- prepareMsg(msg)
	}
}

func (l *logger) Debug(msg string) {
	if l.terminalLogs == true && l.needPrint(Debug) == true {
		fmt.Println(makeMessageColorful(Debug, msg))
	}

	if l.fileLogs == true && l.needWrite(Debug) == true {
		l.debugChan <- prepareMsg(msg)
	}
}

func (l *logger) Query(msg string) {
	//сделать так, чтобы в запросе не было лишних пробелов и табуляций,
	//чтобы аргументы были по значениям, а не указатели
	//prepareQueryMsg(query, args)

	if l.terminalLogs == true && l.needPrint(Query) == true {
		fmt.Println(makeMessageColorful(Query, msg))
	}

	if l.fileLogs == true && l.needWrite(Query) == true {
		l.queryChan <- prepareMsg(msg)
	}
}

//func (l *logger) Critical(msg string) {
//	if l.needPrint(Critical) == true {
//		fmt.Println(makeMessageColorful(Critical, msg))
//	}
//
//	if l.terminalOnly == true {
//		return
//	}
//
//	if l.needWrite(Critical) == true {
//		l.criticalChan <- prepareMsg(msg)
//	}
//}
//
//func (l *logger) Warning(msg string) {
//	if l.needPrint(Warning) == true {
//		fmt.Println(makeMessageColorful(Warning, msg))
//	}
//
//	if l.terminalOnly == true {
//		return
//	}
//
//	if l.needWrite(Warning) == true {
//		l.warningChan <- prepareMsg(msg)
//	}
//}

func (l *logger) Error(msg string) {
	if l.terminalLogs == true && l.needPrint(Error) == true {
		fmt.Println(makeMessageColorful(Error, msg))
	}

	if l.fileLogs == true && l.needWrite(Error) == true {
		l.errorChan <- prepareMsg(msg)
	}
}
