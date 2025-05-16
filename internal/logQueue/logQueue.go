package logQueue

import (
	"log"

	"github.com/Uuq114/JanusLLM/internal/spend"
)

var (
	SpendLogQueue = make(chan spend.SpendRecord, 100)
)

func PushLog(logRecord interface{}, logType string) {
	switch logType {
	case "spend":
		SpendLogQueue <- logRecord.(spend.SpendRecord)
	default:
		log.Fatal("Unknown log type")
	}
}

func FlushLog() {
	var batch []spend.SpendRecord
	for {
		select {
		case logRecord, ok := <-SpendLogQueue:
			if !ok {
				if len(batch) > 0 {
					spend.InsertBatchSpendRecord(batch)
					batch = nil
				}
				return
			}
			batch = append(batch, logRecord)
		default:
			if len(batch) > 0 {
				spend.InsertBatchSpendRecord(batch)
				batch = nil
			}
			return
		}
	}
}
