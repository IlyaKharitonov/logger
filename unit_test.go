package logger

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSortLogs(t *testing.T) {
	list := []msgType{
		{TimeUTC: 99999999},
		{TimeUTC: 11111111},
		{TimeUTC: 55555555},
		{TimeUTC: 77777777},
		{TimeUTC: 66666666},
		{TimeUTC: 33333333},
		{TimeUTC: 12345678},
		{TimeUTC: 67890456},
	}

	fmt.Println(sortLogs(list))

}

func TestAddParam(t *testing.T) {
	config := &LoggerConf{
		PathFolder:     "./logs",
		PrintInfo:      true,
		ChanCapacity:   100,
		BufferCapacity: 10,
		ColorHeadings:  true,
		DebugLog:       false,
	}

	logger := New(config)

	logger.Info("hello world", errors.New("Ошибочка вышла"),
		logger.AddParam("param1", "value1"),
		logger.AddParam("param2", "value2"),
		logger.AddParam("param3", "value3"))

}

// тест логгера на корректную запись и сортировку поступающих логгов
// выясняю не пропадают ли логги при большой нагрузке (40 воркеров, 100000 логов от каждого в каждый канал)
// перед запуском теста не должно быть логов rm -R logs
// обратил внимание что значение лимита (Limit) оптимально в пределах 10-20
// TODO понять как влияет размер буфера в каналах на производительость
// TODO понять как влияет размер слайса в горутине (Limit)
func TestLogger(t *testing.T) {
	//RecordableLevels: []string{Info, Debug, Warning, Critical, Error, Query},

	config := &LoggerConf{
		PathFolder: "./logs",
		//PrintableLevels:  []string{},
		//RecordableLevels: []string{Info, Debug, Error, Query},
		//Limit:            15,
		ColorHeadings: true,
		DebugLog:      false,
	}

	logger := New(config)

	wg := &sync.WaitGroup{}

	//имитируем работы нескольких воркеров
	numWorkers := 40
	numCircles := 100000

	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go workerImitation(i+1, logger, nil, numCircles, wg)
	}

	wg.Wait()

	logger.Stop()

	y, m, d := time.Now().Date()
	partOfName := "_logs_" + strconv.Itoa(d) + "_" + m.String() + "_" + strconv.Itoa(y) + ".log"

	//проверяю количество строк в логе, оно должно быть
	//равно количество горутин на количество циклов внутри горутины
	/////////////////////////////////////////////////////////////////////////////

	dataInfo, _ := ioutil.ReadFile("logs/info/info" + partOfName)
	c := strings.Count(string(dataInfo), "\n")

	if c != numWorkers*numCircles {
		t.Errorf("%v количество строк в info файле %d не равно ожидаемому количеству строк %d %v", red, c, numWorkers*numCircles, noColor)
	} else {
		t.Log("Успех info!!!Строк", c)
	}

	/////////////////////////////////////////////////////////////////////////////
	//уровень убран
	//dataWarning, _ := ioutil.ReadFile("logs/warning/warning" + partOfName)
	//c = strings.Count(string(dataWarning), "\n")
	//
	//if c != numWorkers*numCircles {
	//	t.Errorf("%v количество строк в warning файле %d не равно ожидаемому количеству строк %d %v", red, c, numWorkers*numCircles, noColor)
	//} else {
	//	t.Log("Успех warning!!!Строк", c)
	//}

	/////////////////////////////////////////////////////////////////////////////
	//уровень убран
	//dataCritical, _ := ioutil.ReadFile("logs/critical/critical" + partOfName)
	//c = strings.Count(string(dataCritical), "\n")
	//
	//if c != numWorkers*numCircles {
	//	t.Errorf("%v количество строк в critical файле %d не равно ожидаемому количеству строк %d %v", red, c, numWorkers*numCircles, noColor)
	//} else {
	//	t.Log("Успех critical!!!Строк", c)
	//}

	/////////////////////////////////////////////////////////////////////////////

	dataError, _ := ioutil.ReadFile("logs/error/error" + partOfName)
	c = strings.Count(string(dataError), "\n")

	if c != numWorkers*numCircles {
		t.Errorf("%v количество строк в error файле %d не равно ожидаемому количеству строк %d %v", red, c, numWorkers*numCircles, noColor)
	} else {
		t.Log("Успех error!!!Строк", c)
	}

	/////////////////////////////////////////////////////////////////////////////

	dataDebug, _ := ioutil.ReadFile("logs/debug/debug" + partOfName)
	c = strings.Count(string(dataDebug), "\n")

	if c != numWorkers*numCircles {
		t.Errorf("%v количество строк debug в файле %d не равно ожидаемому количеству строк %d %v", red, c, numWorkers*numCircles, noColor)
	} else {
		t.Log("Успех debug!!!Строк", c)
	}

	/////////////////////////////////////////////////////////////////////////////
	dataQuery, _ := ioutil.ReadFile("logs/query/query" + partOfName)
	c = strings.Count(string(dataQuery), "\n")

	if c != numWorkers*numCircles {
		t.Errorf("%v количество строк query в файле %d не равно ожидаемому количеству строк %d %v", red, c, numWorkers*numCircles, noColor)
	} else {
		t.Log("Успех query!!!Строк", c)
	}

	//cancel()
}

func workerImitation(num int, logger *logger, ctx context.Context, numCircles int, wg *sync.WaitGroup) {
	//имитация работы загрузки логгера в ходе работы воркера
	for i := 1; i <= numCircles; i++ {

		//logger.Info("Запустил сервис. Горутина " + strconv.Itoa(num))
		//logger.Warning("Варнинг. Горутина " + strconv.Itoa(num))
		//logger.Critical("У тебя ПАНИКА. Горутина " + strconv.Itoa(num))
		logger.Error("Ошибка. Горутина " + strconv.Itoa(num))
		logger.Debug("Дебаг. Горутина " + strconv.Itoa(num))
		//logger.Query("Запрос.Горутина " + strconv.Itoa(num))

		//if i == numCircles {
		//	fmt.Println("воркер", num, "отправил "+strconv.Itoa(numCircles)+" логов")
		//}
	}

	wg.Done()
}

//signal.Notify(quit,
//	syscall.SIGTERM, /*  Согласно всякой документации именно он должен останавливать прогу, но на деле его мы не находим. Оставил его просто на всякий случай  */
//	syscall.SIGINT,  /*  Останавливает прогу когда она запущена из терминала и останавливается через CTRL+C  */
//	syscall.SIGQUIT, /*  Останавливает демона systemd  */
//)
//receivedSignal := <-quit
//switch receivedSignal {
//case syscall.SIGTERM:
//	logger.Info(nil, "closing program by SIGTERM (mb reboot of system)")
//case syscall.SIGINT:
//	logger.Info(nil, "closing program by SIGINT (CTRL+C)")
//case syscall.SIGQUIT:
//	logger.Info(nil, "closing program by SIGQUIT (systemd stop)")
//default:
//	//logger.Info(nil, "closing program by default")
//}
