# Scarlet

A lightweight, zero-dependency, local web GUI for chatting with OpenAI and
compatible AI models (learn more about [Chatbots on
Wikipedia](https://wikipedia.org/wiki/Chatbot)). It natively supports
live-streaming, file uploads, and advanced Markdown rendering—all built using
pure Go and standard HTML/CSS with **zero JavaScript**.

## Features

**Zero Dependencies:** Built entirely with the Go standard library.

**No JavaScript:** Uses standard HTML forms and HTTP chunked streaming (SSE) to render live AI responses natively in the browser.

**Custom Markdown Engine:** Safely parses code blocks, ordered/unordered lists, italics, bold, and horizontal rules on the fly.

**Collapsible "Thinking" Blocks:** Natively supports models that return
`<reasoning>` tokens by rendering them into collapsible `<details>` blocks so
they don't clutter your chat history.

**Context-Aware File Uploads:** Upload multiple files, and the server will
automatically convert their contents into formatted Markdown code blocks so the
AI can read them.

## OpenAI

You can manage your OpenAI API keys here:
https://platform.openai.com/api-keys

Configure the application step-by-step. Replace `YOUR_API_KEY` with your actual API key:

```
scarlet -api-key YOUR_API_KEY
scarlet -api-url https://api.openai.com/v1/chat/completions
scarlet -model gpt-4o
```

## Z.ai

You can manage your Z.ai API keys here:
https://z.ai/manage-apikey/apikey-list

You can also find other highly-rated, compatible models on OpenRouter:
https://openrouter.ai/models?order=da-elo-high-to-low&context=64000&max_output_price=40

Configure the application step-by-step. Replace `YOUR_API_KEY` with your actual API key:

```
scarlet -api-key YOUR_API_KEY
scarlet -api-url https://open.bigmodel.cn/api/paas/v4/chat/completions
scarlet -model glm-5.2
```
