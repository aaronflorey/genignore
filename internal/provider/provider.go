package provider

import "context"

type Result struct {
	Key      string `json:"key"`
	Matched  bool   `json:"matched"`
	Reason   string `json:"reason"`
	Evidence string `json:"evidence,omitempty"`
	Error    string `json:"error,omitempty"`
}

type Detector interface {
	Detect(ctx context.Context, cwd string) Result
}
