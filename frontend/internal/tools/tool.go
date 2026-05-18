package tools

import "context"

type Artifact struct {
	Type    string
	Payload any
}

type ParamSpec struct {
	Name string
	Type string
}

type ToolSpec struct {
	Name        string
	Description string
	Params      []ParamSpec
	ReturnType  string
}

type Tool interface {
	Spec() ToolSpec
	Execute(ctx context.Context, tokens []any) (result string, artifacts []Artifact, err error)
}
