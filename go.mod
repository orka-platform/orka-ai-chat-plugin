module orka-ai-chat-plugin

go 1.21

require (
	github.com/openai/openai-go/v2 v2.0.2
	github.com/orka-platform/orka-plugin-sdk v0.0.0
)

require (
	github.com/tidwall/gjson v1.14.4 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
)

replace github.com/orka-platform/orka-plugin-sdk => ./sdk
