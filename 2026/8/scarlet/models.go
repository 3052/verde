package scarlet

type AppConfig struct {
   APIKey string `json:"api_key"`
   APIURL string `json:"api_url"`
   Model  string `json:"model"`
}

type FileAttachment struct {
   Filename string `json:"filename"`
   Content  string `json:"content"`
}

type Message struct {
   Role             string           `json:"role"`
   Content          string           `json:"content"`
   ReasoningContent string           `json:"reasoning_content,omitempty"`
   Files            []FileAttachment `json:"files,omitempty"`
}

type PromptTokensDetails struct {
   CachedTokens int `json:"cached_tokens"`
}

type StreamChoice struct {
   Delta StreamDelta `json:"delta"`
}

type StreamDelta struct {
   Content          string `json:"content"`
   ReasoningContent string `json:"reasoning_content"`
}

type StreamResponse struct {
   Choices []StreamChoice `json:"choices"`
   Usage   *Usage         `json:"usage,omitempty"`
}

type Usage struct {
   PromptTokens        int                 `json:"prompt_tokens"`
   CompletionTokens    int                 `json:"completion_tokens"`
   TotalTokens         int                 `json:"total_tokens"`
   PromptTokensDetails PromptTokensDetails `json:"prompt_tokens_details"`
}

// models.go
