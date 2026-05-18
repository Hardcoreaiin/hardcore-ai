package tools

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

// Register wires a typed handler into a Registry. The args struct T's fields
// become positional parameters in the C-like signature. Use `tool:"name"` and
// `desc:"..."` tags to customize the prompt; default name is the lower-cased
// field name.
//
// The handler returns (resultString, artifacts, error). resultString is what
// the LLM sees as "Tool result: ..."; artifacts go to the event bus.
type Handler[T any] func(ctx context.Context, args T) (string, []Artifact, error)

type reflectTool struct {
	spec    ToolSpec
	argType reflect.Type
	invoke  func(ctx context.Context, argPtr reflect.Value) (string, []Artifact, error)
}

func (t *reflectTool) Spec() ToolSpec { return t.spec }

func (t *reflectTool) Execute(ctx context.Context, tokens []any) (string, []Artifact, error) {
	if len(tokens) > len(t.spec.Params) {
		return "", nil, fmt.Errorf("too many arguments for %s: got %d, want at most %d",
			t.spec.Name, len(tokens), len(t.spec.Params))
	}

	argPtr := reflect.New(t.argType)
	argVal := argPtr.Elem()

	for i, tok := range tokens {
		field := argVal.Field(i)
		if err := assignToken(field, tok); err != nil {
			return "", nil, fmt.Errorf("arg %q: %w", t.spec.Params[i].Name, err)
		}
	}

	return t.invoke(ctx, argPtr)
}

type Registry struct {
	tools map[string]Tool
	order []string
}

func NewRegistry() *Registry {
	return &Registry{tools: map[string]Tool{}}
}

func Register[T any](r *Registry, name, description string, fn Handler[T]) {
	var zero T
	argType := reflect.TypeOf(zero)
	if argType.Kind() != reflect.Struct {
		panic(fmt.Sprintf("tool %s: arg type must be a struct, got %s", name, argType.Kind()))
	}

	params := make([]ParamSpec, argType.NumField())
	for i := 0; i < argType.NumField(); i++ {
		f := argType.Field(i)
		pname := f.Tag.Get("tool")
		if pname == "" {
			pname = strings.ToLower(f.Name)
		}
		params[i] = ParamSpec{Name: pname, Type: goTypeName(f.Type)}
	}

	tool := &reflectTool{
		spec: ToolSpec{
			Name:        name,
			Description: description,
			Params:      params,
			ReturnType:  "str",
		},
		argType: argType,
		invoke: func(ctx context.Context, argPtr reflect.Value) (string, []Artifact, error) {
			args := argPtr.Elem().Interface().(T)
			return fn(ctx, args)
		},
	}

	if _, dup := r.tools[name]; !dup {
		r.order = append(r.order, name)
	}
	r.tools[name] = tool
}

func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) All() []Tool {
	out := make([]Tool, 0, len(r.order))
	for _, n := range r.order {
		out = append(out, r.tools[n])
	}
	return out
}

func (r *Registry) Names() []string {
	out := make([]string, len(r.order))
	copy(out, r.order)
	return out
}

func goTypeName(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "str"
	case reflect.Bool:
		return "bool"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "int"
	case reflect.Float32, reflect.Float64:
		return "float"
	default:
		return "str"
	}
}

func assignToken(field reflect.Value, tok any) error {
	if tok == nil {
		field.SetZero()
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(toString(tok))
	case reflect.Bool:
		b, ok := tok.(bool)
		if !ok {
			return fmt.Errorf("expected bool, got %T (%v)", tok, tok)
		}
		field.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := toInt(tok)
		if err != nil {
			return err
		}
		field.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, err := toInt(tok)
		if err != nil {
			return err
		}
		if i < 0 {
			return fmt.Errorf("expected unsigned int, got %d", i)
		}
		field.SetUint(uint64(i))
	case reflect.Float32, reflect.Float64:
		f, err := toFloat(tok)
		if err != nil {
			return err
		}
		field.SetFloat(f)
	default:
		return fmt.Errorf("unsupported field kind %s", field.Kind())
	}
	return nil
}

func toString(tok any) string {
	switch v := tok.(type) {
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func toInt(tok any) (int64, error) {
	switch v := tok.(type) {
	case int64:
		return v, nil
	case float64:
		return int64(v), nil
	default:
		return 0, fmt.Errorf("expected int, got %T (%v)", tok, tok)
	}
}

func toFloat(tok any) (float64, error) {
	switch v := tok.(type) {
	case float64:
		return v, nil
	case int64:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("expected float, got %T (%v)", tok, tok)
	}
}
