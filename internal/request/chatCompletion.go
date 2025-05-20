package request

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatReqBody struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	DoSample    bool      `json:"do_sample" default:"true"`
	Temperature float64   `json:"temperature" default:"0.7"`
	TopP        float64   `json:"top_p" default:"1.0"`
	MaxTokens   int       `json:"max_tokens" default:"4096"`
	Stream      bool      `json:"stream" default:"false"`
}
