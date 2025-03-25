package logger

import (
	"fmt"
	"log"
	"os"
	"path"
	"sort"
	"strconv"
	"time"
)

func (l *logger) listenChan(level string) {
	defer l.wg.Done()
	ch := l.getChan(level)
	logs := make([]msgType, 0, l.bufferCapacity)

	//добавление логов в файл происходит пачками равными размеру массива logs
	//экспериментальным путем выяснил, что эффективнее всего иметь размер такой пачки примерно 10-20 логов
	//если в пачке меньше 10 логов мы дрочим лишний раз открытие файла (обращение к системе)
	//если в пачке более 20 логов, например 1000, то получается слишком большой кусок данных,
	//который долго проходит этапы подготовки и долго записывается

	after := time.After(time.Second * 20)

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
				logs = make([]msgType, 0, l.bufferCapacity)
			}
			//перезапускаю таймер
			after = time.After(time.Second * 20)

		//обычный сценарий который срабатывает при заполненности буфера
		case log := <-ch:
			if len(logs) == l.bufferCapacity {
				l.debug(fmt.Sprintf("%sсохраняю логги из слайса канала %s%s", red, level, noColor))
				l.write(level, logs)
				logs = make([]msgType, 0, l.bufferCapacity)
			}

			logs = append(logs, log)
			//default:
		}
	}
}

/*сохраняет оставшиеся логи из канала и слайса перед завершением работы*/
func (l *logger) saveBeforeExit(ch chan msgType, level string, logs []msgType) {
	//сценарий если в слайсе на момент высова метода уже успели набежать логи
	if len(logs) != 0 {
		l.write(level, logs)
	}

	//сценарий если в канале остались необработанные сообщения от воркеров
	if len(ch) > 0 {
		l.saveFromChannel(ch, level)
	}
}

func (l *logger) saveFromChannel(ch chan msgType, level string) {
	logs := make([]msgType, 0, l.bufferCapacity)

	//добавление логов в файл происходит порционно пачками равными размеру массива logs
	//экспериментальным путем выяснил, что эффективнее всего иметь размер такой пачки примерно 10-20 логов
	//если в пачке меньше 10 логов мы дрочим лишний раз открытие файла (обращение к системе)
	//если в пачке более 20 логов, например 1000, то получается слишком большой кусок данных,
	//который долго проходит этапы подготовки и долго записывается
	for log := range ch {
		//1 этап. если слайс заполнен, то записываем содержимое в файл и обнуляем массив
		if len(logs) == l.bufferCapacity {
			l.write(level, logs)
			logs = make([]msgType, 0, l.bufferCapacity)
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

func (l *logger) write(level string, msgList []msgType) {
	var (
		fileName    = getFileName(level)
		msgByte     = l.prepareMsgB(msgList)
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
func (l *logger) prepareMsgB(msgList []msgType) []byte {
	var (
		msgStr        string
		sortedMsgList = sortLogs(msgList)
	)

	//fmt.Println(darkPurple, sortedMsgList, noColor)

	for _, m := range sortedMsgList {
		t := strconv.Itoa(int(m.TimeUTC))

		var msg string
		if l.colorHeadings == true {
			msg = darkGreen + "time: " + noColor + t + ", " + darkGreen + "message: " + noColor + m.Msg + "\n"
		} else {
			msg = "time: " + t + ", " + "message: " + m.Msg + "\n"
		}

		msgStr = msgStr + msg
	}

	return []byte(msgStr)
}

// сортирует логги из канала. отдает упорядоченный по таймштампу массив логгов
func sortLogs(msgList []msgType) []msgType {
	sort.Slice(msgList, func(i, j int) (less bool) {
		return msgList[i].TimeUTC < msgList[j].TimeUTC
	})

	return msgList
}

// проверяет заполненность канала. если канал заполнен до лимита, то вернет true
func checkOccupancyChan(logChan chan msgType, limit int) bool {
	if len(logChan) >= limit {
		return true
	}

	return false
}

func prepareMsg(msgText string) msgType {
	return msgType{
		TimeUTC: time.Now().Unix(),
		Msg:     msgText,
	}
}

func makeMessageColorful(level, msg string) string {
	switch level {
	case Info:
		return darkGreen + "\nLevel: " + level + noColor + msg
	case Debug:
		return blue + level + ": " + noColor + msg
	case Error:
		return red + level + ": " + noColor + msg
	//case Query:
	//	return orange + level + ": " + noColor + msg
	//case Critical:
	//	return darkBlue + level + ": " + noColor + msg
	//case Warning:
	//	return darkPurple + level + ": " + noColor + msg
	default:
		return msg
	}
}

func (l *logger) getChan(level string) chan msgType {
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
		fmt.Println(msg)
	}
}
