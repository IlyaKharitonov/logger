package logger

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
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
	errorChan chan msgType

	bufferCapacity int //предел заполненности слайса, после которого логги из него будут считаны и отправлены на сортировку
	chanCapacity   int

	printInfo  bool
	printError bool
	printDebug bool

	writeInfo  bool
	writeError bool
	writeDebug bool

	pathFolder    string
	colorHeadings bool //вкл/выкл покраску заголовков в файлах логов. покрашенные заголовки упрощают чтение лога из консоли
	//terminalLogs  bool
	//fileLogs      bool

	//stop    bool
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
		infoChan:  make(chan msgType, config.ChanCapacity),
		debugChan: make(chan msgType, config.ChanCapacity),
		errorChan: make(chan msgType, config.ChanCapacity),

		printInfo:  config.PrintInfo,
		printError: config.PrintError,
		printDebug: config.PrintDebug,

		writeInfo:  config.WriteInfo,
		writeError: config.WriteError,
		writeDebug: config.WriteDebug,

		pathFolder:     config.PathFolder,
		bufferCapacity: config.BufferCapacity,

		colorHeadings: config.ColorHeadings,

		wg:       wg,
		stopped:  make(chan struct{}),
		ctx:      ctx,
		cancel:   cancel,
		debugLog: config.DebugLog,
	}

	if logger.writeError == true || logger.writeInfo == true || logger.writeDebug == true {
		go logger.StartProcessingLogs()
	}

	return logger
}

// читает из каналов логи и пишет их в слайсы, для дальнейшей обработки и записи
func (l *logger) StartProcessingLogs() {
	/*пуск горутин на каждый уровень логирования, указанный в конфигурации*/
	if l.writeInfo == true {
		l.debug(fmt.Sprintf("пуск горутины для канала %s", Info))

		l.wg.Add(1)
		go l.listenChan(Info)
	}

	if l.writeDebug == true {
		l.debug(fmt.Sprintf("пуск горутины для канала %s", Debug))

		l.wg.Add(1)
		go l.listenChan(Debug)
	}

	if l.writeError == true {
		l.debug(fmt.Sprintf("пуск горутины для канала %s", Error))

		l.wg.Add(1)
		go l.listenChan(Error)
	}

	l.debug("жду в вызывающей горутине")

	l.wg.Wait()

	l.debug("отправляю сигнал о выполненной остановке в вызывающей горутине")

	l.stopped <- struct{}{}
}

// Stop() graceful stop
func (l *logger) Stop() {
	if l.writeError == false && l.writeInfo == false && l.writeDebug == false {
		return
	}

	l.debug("отправляю сигнал на остановку")
	l.cancel()
	l.debug("жду завершения работы горутин")
	<-l.stopped
	l.debug("логгер завершил работу")

}

type arg struct {
}

func (l *logger) AddParam(key string, value interface{}) string {
	return key + "=" + fmt.Sprint(value)
}

func (l *logger) Info(msg string, err error, params ...string) {
	now := time.Now()
	h, m, s := now.Clock()
	y, month, d := now.Date()

	msg = "\nData: " + strconv.Itoa(d) + "." + strconv.Itoa(int(month)) + "." + strconv.Itoa(y) + " " +
		strconv.Itoa(h) + ":" + strconv.Itoa(m) + ":" + strconv.Itoa(s) + " " +
		"\nMessage: " + msg + "\nParams: " + strings.Join(params, ", ")

	if err != nil {
		msg += "\nError: " + err.Error()
	}

	if l.printInfo == true {
		fmt.Println(makeMessageColorful(Info, msg))
	}

	if l.writeInfo == true {
		l.infoChan <- prepareMsg(msg)
	}
}

//func (l *logger) Info(msg string) {
//	if l.printInfo == true {
//		fmt.Println(makeMessageColorful(Info, msg))
//	}
//
//	if l.writeInfo == true {
//		l.infoChan <- prepareMsg(msg)
//	}
//}

func (l *logger) Debug(msg string) {
	if l.printDebug == true {
		fmt.Println(makeMessageColorful(Debug, msg))
	}

	if l.writeDebug == true {
		l.debugChan <- prepareMsg(msg)
	}
}

func (l *logger) Error(msg string) {
	if l.printError == true {
		fmt.Println(makeMessageColorful(Error, msg))
	}

	if l.writeError == true {
		l.errorChan <- prepareMsg(msg)
	}
}
