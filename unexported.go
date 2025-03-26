package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (l *logger) listenChan(level string) {
	defer l.wg.Done()
	ch := l.getChan(level)
	logs := make([]*recordType, 0, l.bufferCapacity)

	//добавление логов в файл происходит пачками равными размеру массива logs
	//экспериментальным путем выяснил, что эффективнее всего иметь размер такой пачки примерно 10-20 логов
	//если в пачке меньше 10 логов мы дрочим лишний раз открытие файла (обращение к системе)
	//если в пачке более 20 логов, например 1000, то получается слишком большой кусок данных,
	//который долго проходит этапы подготовки и долго записывается

	after := time.After(time.Second * time.Duration(int(l.writeTimout)))

	for {
		select {
		//сценарий сохранения логов после сигнала остановки
		case <-l.ctx.Done():
			l.debug(fmt.Sprintf("%sзапускаю сохранение перед остановкой %s. количество несохраненных логов в канале %v%s", darkGreen, level, len(ch), noColor))
			l.saveBeforeExit(ch, level, logs)
			l.debug(fmt.Sprintf("%sзавершил сохранение перед остановкой, перестал слушать канал %s%s", darkBlue, level, noColor))
			return

		//сценарий когда долго не поступало логов и буфер logs полупустой
		case <-after:
			if len(logs) > 0 {
				l.debug(fmt.Sprintf("%sсохраняю логги из полупустого слайса канала%s%s", darkPurple, level, noColor))
				l.write(level, logs)
				logs = make([]*recordType, 0, l.bufferCapacity)
			}
			//перезапускаю таймер
			after = time.After(time.Second * time.Duration(int(l.writeTimout)))

		//обычный сценарий который срабатывает при заполненности буфера
		case log := <-ch:
			if len(logs) == l.bufferCapacity {
				l.debug(fmt.Sprintf("%sсохраняю логги из слайса канала %s%s", red, level, noColor))
				l.write(level, logs)
				logs = make([]*recordType, 0, l.bufferCapacity)
			}

			logs = append(logs, log)
			//default:
		}
	}
}

/*сохраняет оставшиеся логи из канала и слайса перед завершением работы*/
func (l *logger) saveBeforeExit(ch chan *recordType, level string, logs []*recordType) {
	//сценарий если в слайсе на момент высова метода уже успели набежать логи
	if len(logs) != 0 {
		l.write(level, logs)
	}

	//сценарий если в канале остались необработанные сообщения от воркеров
	if len(ch) > 0 {
		l.saveFromChannel(ch, level)
	}
}

func (l *logger) saveFromChannel(ch chan *recordType, level string) {
	logs := make([]*recordType, 0, l.bufferCapacity)

	//добавление логов в файл происходит порционно пачками равными размеру массива logs
	//экспериментальным путем выяснил, что эффективнее всего иметь размер такой пачки примерно 10-20 логов
	//если в пачке меньше 10 логов мы дрочим лишний раз открытие файла (обращение к системе)
	//если в пачке более 20 логов, например 1000, то получается слишком большой кусок данных,
	//который долго проходит этапы подготовки и долго записывается
	for log := range ch {
		//1 этап. если слайс заполнен, то записываем содержимое в файл и обнуляем массив
		if len(logs) == l.bufferCapacity {
			l.write(level, logs)
			logs = make([]*recordType, 0, l.bufferCapacity)
		}

		//2 этап.
		logs = append(logs, log)

		//3 этап. Если все по нулям после 1 этапа, то выходим
		if len(logs) == 0 && len(ch) == 0 {
			break
		}

		//4 этап. Записываем остатки из слайса logs, которых не хватило,
		//чтобы заполнить буфер до предела и вызвать 1 этап
		if len(ch) == 0 && len(logs) != 0 {
			l.write(level, logs)
			break
		}
	}
}

func getFileName(level string) string {
	y, m, d := time.Now().Date()
	return level + "_logs_" + strconv.Itoa(d) + "_" + m.String() + "_" + strconv.Itoa(y) + ".log"
}

func (l *logger) write(level string, recordList []*recordType) {
	var (
		fileName    = getFileName(level)
		msgByte     = l.prepareRecordByte(recordList)
		directories = path.Join(l.pathFolder, level)
		pathFile    = path.Join(directories, fileName)
	)

	err := os.MkdirAll(directories, 0777)
	if err != nil {
		log.Fatal("Создать директории не удалось ", err)
	}

	file, err := os.OpenFile(pathFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		log.Fatal("Открыть файл не удалась ", err)
	}
	defer file.Close()

	_, err = file.Write(msgByte)
	if err != nil {
		log.Fatal("Запись файла не удалась ", err)
	}
}

// подготавливает список логов к записи
func (l *logger) prepareRecordByte(recordList []*recordType) []byte {

	sortedRecordList := sortLogs(recordList)

	if l.format == JSONFormat {
		return l.prepareJSON(sortedRecordList)
	}

	return l.prepareString(sortedRecordList)

}

func (l *logger) prepareJSON(recordList []*recordType) []byte {
	var list []string

	for _, r := range recordList {
		json, err := json.Marshal(r)
		if err != nil {
			log.Fatal("(l *logger) prepareJSON ", err)
		}

		if len(list) == 0 {
			list = append(list, "\n")
		}

		list = append(list, string(json))
	}

	return []byte(strings.Join(list, "\n"))
}

func (l *logger) prepareString(recordList []*recordType) []byte {
	var list []string

	for _, r := range recordList {
		recordString :=
			"Level: " + r.Level +
				", Date: " + r.Date +
				", Message: " + r.Message

		if len(r.Params) != 0 {
			recordString += ", Params: " + strings.Join(r.Params, ", ")
		}

		if r.Error != nil {
			recordString += ", Error: " + *r.Error
		}

		if len(list) == 0 {
			list = append(list, "\n")
		}

		list = append(list, recordString)
	}

	return []byte(strings.Join(list, "\n"))
}

// сортирует логги из канала. отдает упорядоченный по таймштампу массив логгов
func sortLogs(recordList []*recordType) []*recordType {
	sort.Slice(recordList, func(i, j int) (less bool) {
		return recordList[i].TimeUTC < recordList[j].TimeUTC
	})

	return recordList
}

// проверяет заполненность канала. если канал заполнен до лимита, то вернет true
func checkOccupancyChan(logChan chan recordType, limit int) bool {
	if len(logChan) >= limit {
		return true
	}

	return false
}

func (l *logger) collectRecord(level string, msg string, err error, params ...string) *recordType {
	now := time.Now()
	date := now.Format("02.01.2006 15:04:05")

	record := &recordType{
		TimeUTC: now.Unix(),
		Level:   level,
		Date:    date,
		Message: msg,
		Params:  params,
	}

	if err != nil {
		errStr := err.Error()
		record.Error = &errStr
	}

	return record
}

func (l *logger) prepareToPrint(record *recordType) string {
	if l.colorHeadings == true {
		return makeMessageColorful(record)
	}

	recordString :=
		"\nLevel: " + record.Level +
			"\nDate: " + record.Date +
			"\nMessage: " + record.Message

	if record.Error != nil {
		recordString += "\nError: " + *record.Error
	}

	return recordString
}

func makeMessageColorful(record *recordType) string {
	var color string

	switch record.Level {
	case Info:
		color = darkGreen
	case Debug:
		color = blue
	case Error:
		color = red
	//case Query:
	//	return orange + level + ": " + noColor + msg
	//case Critical:
	//	return darkBlue + level + ": " + noColor + msg
	//case Warning:
	//	return darkPurple + level + ": " + noColor + msg
	default:
	}

	recordString :=
		"\nLevel: " + color + record.Level + noColor +
			"\nDate: " + record.Date +
			"\nMessage: " + record.Message

	if len(record.Params) != 0 {
		recordString += "\nParams: " + strings.Join(record.Params, ", ")
	}

	if record.Error != nil {
		recordString += "\nError: " + *record.Error
	}

	return recordString
}

func (l *logger) getChan(level string) chan *recordType {
	switch level {
	case Info:
		return l.infoChan
	case Debug:
		return l.debugChan
	case Error:
		return l.errorChan
	//case Query:
	//	return l.queryChan
	//case Critical:
	//	return l.criticalChan
	//case Warning:
	//	return l.warningChan

	default:
		return nil
	}
}

func (l *logger) debug(msg string) {
	if l.debugLog == true {
		fmt.Println("Дебагер логгера: ", msg)
	}
}
