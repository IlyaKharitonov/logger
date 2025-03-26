package logger

import (
	"context"
	"fmt"
	"log"
	"sync"
)

type ILogger interface {
	Info(msg string)  //Информационные сообщения о ходе работы программы
	Debug(msg string) //Сообщения отладки
	Error(msg string) //Ошибка в ходе работы программы

	Stop()
}

type logger struct {
	infoChan  chan *recordType
	debugChan chan *recordType
	errorChan chan *recordType

	bufferCapacity int //предел заполненности слайса, после которого логги из него будут считаны и отправлены на сортировку
	chanCapacity   int

	printInfo  bool
	printError bool
	printDebug bool

	writeInfo  bool
	writeError bool
	writeDebug bool

	format        string
	writeTimout   uint
	withoutWrite  bool
	pathFolder    string
	colorHeadings bool //вкл/выкл покраску заголовков в файлах логов. покрашенные заголовки упрощают чтение лога из консоли

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

	if config.Format != JSONFormat && config.Format != TextFormat {
		log.Fatal("Поле Format должно содержать 'text' или 'json'")
	}

	if config.BufferCapacity == 0 {
		log.Fatal("Поле BufferCapacity должно быть больше нуля")
	}

	if config.ChanCapacity == 0 {
		log.Fatal("Поле ChanCapacity должно быть больше нуля")
	}

	logger := &logger{
		infoChan:  make(chan *recordType, config.ChanCapacity),
		debugChan: make(chan *recordType, config.ChanCapacity),
		errorChan: make(chan *recordType, config.ChanCapacity),

		printInfo:  config.PrintInfo,
		printError: config.PrintError,
		printDebug: config.PrintDebug,

		writeInfo:  config.WriteInfo,
		writeError: config.WriteError,
		writeDebug: config.WriteDebug,

		writeTimout:    config.WriteTimout,
		format:         config.Format,
		pathFolder:     config.PathFolder,
		bufferCapacity: config.BufferCapacity,
		colorHeadings:  config.ColorHeadings,

		wg:       wg,
		stopped:  make(chan struct{}),
		ctx:      ctx,
		cancel:   cancel,
		debugLog: config.DebugLog,
	}

	if logger.writeError == false && logger.writeInfo == false && logger.writeDebug == false {
		logger.withoutWrite = true
	}

	if logger.withoutWrite == false {
		go logger.startProcessingLogs()
	}

	return logger
}

// читает из каналов логи и пишет их в слайсы, для дальнейшей обработки и записи
func (l *logger) startProcessingLogs() {
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
	if l.withoutWrite == true {
		return
	}

	l.debug("отправляю сигнал на остановку")
	l.cancel()
	l.debug("жду завершения работы горутин")
	<-l.stopped
	l.debug("логгер завершил работу")
}

func (l *logger) AddParam(key string, value interface{}) string {
	return key + "=" + fmt.Sprint(value)
}

func (l *logger) Info(msg string, err error, params ...string) {
	record := l.collectRecord(Info, msg, err, params...)

	if l.printInfo == true {
		fmt.Println(l.prepareToPrint(record))
	}

	if l.writeInfo == true {
		l.infoChan <- record
	}
}

func (l *logger) Debug(msg string, err error, params ...string) {
	record := l.collectRecord(Debug, msg, err, params...)

	if l.printDebug == true {
		fmt.Println(l.prepareToPrint(record))
	}

	if l.writeDebug == true {
		l.debugChan <- record
	}
}

func (l *logger) Error(msg string, err error, params ...string) {
	record := l.collectRecord(Error, msg, err, params...)

	if l.printError == true {
		fmt.Println(l.prepareToPrint(record))
	}

	if l.writeError == true {
		l.errorChan <- record
	}
}
