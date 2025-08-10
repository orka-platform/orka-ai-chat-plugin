package sdk

// Request is the RPC input for Orka plugins.
// Method is a case-sensitive logical operation name.
// Args carries method-specific parameters.
type Request struct {
	Method string         `json:"method"`
	Args   map[string]any `json:"args"`
}

// Response is the RPC output for Orka plugins.
// Data is either a map or scalar. If scalar, the engine wraps it as {"result": <value>}.
type Response struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Data    any    `json:"data,omitempty"`
}
