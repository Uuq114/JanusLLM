package logQueue

import "github.com/Uuq114/JanusLLM/internal/spend"

var (
	SpendLogQueue = make(chan interface{}, 100)
)

func PushLogQueue(log interface{}, logType string) {
	switch logType {
	case "spend":
		SpendLogQueue <- log
	default:
		log.Fatal("Unknown log type")
	}
}

func FlushLogQueue() {
	for _, log := range SpendLogQueue {
		switch log.(type) {
		case spend.SpendRecord:
			spend.CreateSpendRecord(log.(spend.SpendRecord).RequestId, log.(spend.SpendRecord).AuthKey,
				log.(spend.SpendRecord).ModelGroup, log.(spend.SpendRecord).TotalTokens,
				log.(spend.SpendRecord).PromptTokens, log.(spend.SpendRecord).CompletionTokens,
				log.(spend.SpendRecord).CreateTime)
		default:
			log.Fatal("Unknown log type")
		}
	}
}
