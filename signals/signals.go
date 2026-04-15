package signals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	log "log/slog"
	"net/http"
	"regexp"
	"strings"
)

// Expr is a JavaScript/Datastar expression string. Its underlying type is
// string, satisfying templ's stringable constraint (~string), so it can be
// used directly in templ attribute interpolation without calling .String():
//
//	data-show={ errMsg.NotEmpty() }
//	data-attr:disabled={ submitting }
//	data-text={ submitting.Then("Saving...", "Submit") }
type Expr string

// String satisfies fmt.Stringer.
func (e Expr) String() string { return string(e) }

func (e Expr) unaryOp(op string) Expr {
	return Expr(fmt.Sprintf("(%s %s)", op, e))
}

func (e Expr) binaryOp(op string, rhs any) Expr {
	return Expr(fmt.Sprintf("(%s %s %s)", e, op, exprVal(rhs)))
}

// func (e Expr) Dot(prop string) Expr { return Expr(fmt.Sprintf("%s.%s", exprVal(e), prop)) }

//// Logical

func (e Expr) Not() Expr        { return e.unaryOp("!") }
func (e Expr) And(rhs any) Expr { return e.binaryOp("&&", rhs) }
func (e Expr) Or(rhs any) Expr  { return e.binaryOp("||", rhs) }

//// Comparison

func (e Expr) Eq(rhs any) Expr   { return e.binaryOp("==", rhs) }
func (e Expr) Neq(rhs any) Expr  { return e.binaryOp("!=", rhs) }
func (e Expr) Gt(rhs any) Expr   { return e.binaryOp(">", rhs) }
func (e Expr) GtEq(rhs any) Expr { return e.binaryOp(">=", rhs) }
func (e Expr) Lt(rhs any) Expr   { return e.binaryOp("<", rhs) }
func (e Expr) LtEq(rhs any) Expr { return e.binaryOp("<=", rhs) }

//// Arithmetic

func (e Expr) Add(rhs any) Expr { return e.binaryOp("+", rhs) }
func (e Expr) Sub(rhs any) Expr { return e.binaryOp("-", rhs) }
func (e Expr) Mul(rhs any) Expr { return e.binaryOp("*", rhs) }
func (e Expr) Div(rhs any) Expr { return e.binaryOp("/", rhs) }
func (e Expr) Mod(rhs any) Expr { return e.binaryOp("%", rhs) }

//// Bitwise

func (e Expr) BitOr(rhs any) Expr  { return e.binaryOp("|", rhs) }
func (e Expr) BitAnd(rhs any) Expr { return e.binaryOp("&", rhs) }
func (e Expr) BitXor(rhs any) Expr { return e.binaryOp("^", rhs) }
func (e Expr) Shl(rhs any) Expr    { return e.binaryOp("<<", rhs) }
func (e Expr) Shr(rhs any) Expr    { return e.binaryOp(">>", rhs) }

//// Other Expressions

// Produces a ternary expression.
func (e Expr) IfThen(a, b any) Expr {
	return Expr(fmt.Sprintf("(%s ? %s : %s)", e, exprVal(a), exprVal(b)))
}

func (e Expr) Inc() Expr { return e.Set(e.Add(+1)) }

func (e Expr) Dec() Expr { return e.Set(e.Add(-1)) }

func (e Expr) Set(rhs any) Expr { return e.binaryOp("=", rhs) }

func (e Expr) Toggle() Expr { return e.Set(e.Not()) }

//// String

// Empty checks whether the expression equals an empty string.
//
//	email.Empty() // "($email == '')"
func (e Expr) Empty() Expr { return e.Eq(Expr("''")) }

// NotEmpty checks whether the expression is a non-empty string.
//
//	errMsg.NotEmpty() // "($errMsg != '')"
func (e Expr) NotEmpty() Expr { return e.Neq(Expr("''")) }

//// Backend actions

// Post produces a Datastar @post() action expression.
func (e Expr) Post(url string) Expr { return Expr(fmt.Sprintf("@post('%s')", url)) }

// Get produces a Datastar @get() action expression.
func (e Expr) Get(url string) Expr { return Expr(fmt.Sprintf("@get('%s')", url)) }

// Put produces a Datastar @put() action expression.
func (e Expr) Put(url string) Expr { return Expr(fmt.Sprintf("@put('%s')", url)) }

// Patch produces a Datastar @patch() action expression.
func (e Expr) Patch(url string) Expr { return Expr(fmt.Sprintf("@patch('%s')", url)) }

// Delete produces a Datastar @delete() action expression.
func (e Expr) Delete(url string) Expr { return Expr(fmt.Sprintf("@delete('%s')", url)) }

// Post produces a Datastar @post() action expression.
func DataPost(url string) Expr { return Expr(fmt.Sprintf("@post('%s')", url)) }

// Get produces a Datastar @get() action expression.
func DataGet(url string) Expr { return Expr(fmt.Sprintf("@get('%s')", url)) }

// Put produces a Datastar @put() action expression.
func DataPut(url string) Expr { return Expr(fmt.Sprintf("@put('%s')", url)) }

// Patch produces a Datastar @patch() action expression.
func DataPatch(url string) Expr { return Expr(fmt.Sprintf("@patch('%s')", url)) }

// Delete produces a Datastar @delete() action expression.
func DataDelete(url string) Expr { return Expr(fmt.Sprintf("@delete('%s')", url)) }

// UnsafeRawExpr creates an Expr from a raw JavaScript/Datastar expression string.
// Use for expressions that don't correspond to a single signal path.
//
// SECURITY: Never pass user-controlled input to UnsafeRawExpr()  it is embedded
// verbatim into JS expressions without escaping. Use the typed operator
// methods (Eq, Then, etc.) for user-derived values instead.
//
//	UnsafeRawExpr("true")
//	UnsafeRawExpr("$count > 0")
func UnsafeRawExpr(raw string) Expr {
	return Expr(raw)
}

// Signal is a namespaced reference to a single named signal in the Datastar signal tree.
// It embeds Expr, so all expression operators are available directly.
type Signal struct {
	Expr        // $ prefixed expression; satisfies templ's stringable via ~string
	path string // bare path without $; for Name() only; empty on derived expressions
}

func (ns Signals) Derive(parts ...string) Signals {
	if ns.prefix == "" {
		return Signals{prefix: strings.Join(parts, ".")}
	}
	s := Signals{prefix: ns.prefix + "." + strings.Join(parts, ".")}
	return s
}

// Scope replaces `#s.` with the signal namespace, and also supports
// named placeholders for Go values via pairs of ("placeholder", value).
// Placeholders are referenced as #placeholder in the expression — they must
// be valid JS identifiers (letter or underscore, then letters/digits/underscores).
//
//	Example: s.Scope(`#s.order_by == #field`, "field", col.Field)
//	Becomes: "($forms.table.order_by == 'products')"
//
// NOTE: Scope is mainly for complex-but-readable logic. Anything beyond
// 4-5 expressions should be a JavaScript function instead.
func (ns Signals) Scope(raw string, substitutions ...any) Expr {
	if len(substitutions)%2 != 0 {
		panic("signals: Scope substitutions must be key-value pairs")
	}

	subs := make(map[string]string, len(substitutions)/2)
	for i := 0; i < len(substitutions); i += 2 {
		key, ok := substitutions[i].(string)
		// TODO?: Place this validation behind a compile time constant
		if !ok {
			panic(fmt.Sprintf("signals: Scope substitution key must be a string, got %T", substitutions[i]))
		}
		if !validPlaceholder(key) {
			panic(fmt.Sprintf("signals: invalid placeholder %q: must be a valid JS identifier", key))
		}
		subs[key] = exprVal(substitutions[i+1])
	}

	prefix := "$" + ns.prefix + "."
	raw = strings.TrimSpace(raw)

	var b strings.Builder
	b.Grow(len(raw) + 64) // Preallocate to reduce chances of needing reallocation

	i := 0
	for i < len(raw) {
		if raw[i] != '#' {
			b.WriteByte(raw[i])
			i++
			continue
		}

		lookahead := i + 1
		for lookahead < len(raw) && validPlaceholderPart(raw[lookahead]) {
			lookahead++
		}

		identifier := raw[i+1 : lookahead]

		switch {
		case identifier == "s" && lookahead < len(raw) && raw[lookahead] == '.':
			b.WriteString(prefix)
			i = lookahead + 1 // skip past `#s.`

		case subs[identifier] != "":
			b.WriteString(subs[identifier])
			i = lookahead

		default:
			// Unknown # token, emit as-is
			b.WriteByte('#')
			i++
		}
	}

	return Expr(b.String())
}

func validPlaceholder(s string) bool {
	if len(s) == 0 {
		return false
	}

	for _, r := range s {
		if !validPlaceholderPart(byte(r)) {
			return false
		}
	}
	return true
}

func validPlaceholderPart(c byte) bool {
	return (c == '_') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// Name returns the bare signal path without the $ prefix.
// Only meaningful on signals returned directly by Signals.Get.
// Derived expressions (results of chaining) return "".
//
// Use with: data-bind, data-indicator, data-computed
//
//	<input data-bind={ email.Name() } />
//	<button data-indicator={ loading.Name() }></button>
func (s Signal) Name() string {
	return s.path
}

//// Signals

// Signals represents a namespace in the Datastar signal tree.
//
//	s := New("forms", "contact_1")
//	s.Get("email").Name()   // "forms.contact_1.email"
//	s.Get("email").String() // "$forms.contact_1.email"
type Signals struct {
	prefix string
}

// Note that we disallow uppercase, that's because attributes are case insensitive in HTML and datastar already does case conversions so it's not correct to do so
var validPathPart = regexp.MustCompile(`^[a-z0-9_-]+$`)

// New creates a new Signals namespace from one or more path parts.
// Panics if any part contains characters outside [a-z0-9_].
//
// Use only with string literals or constants. never with user-supplied input.
// For runtime/user-derived parts, use NewSignalsFromInput which returns an error.
//
//	New("forms", "billing")   // prefix: "forms.billing"
//	New("modals", "confirm")  // prefix: "modals.confirm"
func New(parts ...string) Signals {
	for _, p := range parts {
		if !validPathPart.MatchString(p) {
			panic(fmt.Sprintf("signals: invalid namespace part %q: must match [a-z0-9_]+", p))
		}
	}
	return Signals{prefix: strings.Join(parts, ".")}
}

// NewFromInput creates a new Signals namespace from runtime or
// user-derived path parts. Returns an error instead of panicking, making
// it safe to use with data from requests, DB IDs, etc.
//
//	s, err := NewFromInput("forms", userID)
func NewFromInput(parts ...string) (Signals, error) {
	for _, p := range parts {
		if !validPathPart.MatchString(p) {
			return Signals{}, fmt.Errorf("signals: invalid namespace part %q: must match [a-z0-9_]+", p)
		}
	}
	return Signals{prefix: strings.Join(parts, ".")}, nil
}

// Get a Signal for the given field within this namespace.
func (ns Signals) Get(field string) Signal {
	path := field
	if ns.prefix != "" {
		path = ns.prefix + "." + field
	}

	return Signal{Expr: Expr("$" + path), path: path}
}

type signalContextKey struct{}

var ContextSignalKey = signalContextKey{}

var ErrNoSignalInContext = errors.New("signals not present in context")

// ReadFromContext reads signal bytes previously stored by SignalMiddleware and unmarshals them into T.
func ReadFromContext[T any](ns Signals, ctx context.Context) (T, error) {
	if data, ok := ctx.Value(ContextSignalKey).([]byte); ok {
		return ReadFromBytes[T](ns, data)
	}
	var zero T
	return zero, ErrNoSignalInContext
}

// Dynamically extract a signal from request. Note that this will exhaust the request body reader. For multiple reads, use
func Read[T any](ns Signals, r *http.Request) (out T, err error) {
	switch r.Method {
	case http.MethodGet:
		return ReadFromQuery[T](ns, r)
	case http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete:
		return ReadFromBody[T](ns, r)
	default:
		return out, fmt.Errorf("unsupported method for signal reading")
	}
}

// Dynamically extract a signal from request body
func ReadFromBody[T any](ns Signals, r *http.Request) (out T, err error) {
	data, err := ReadWholeBody(r)
	if err != nil {
		return out, err
	}

	return ReadFromBytes[T](ns, data)
}

func ReadWholeBody(r *http.Request) ([]byte, error) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return data, err
	}
	defer r.Body.Close()
	return data, err
}

func ReadFromQuery[T any](ns Signals, r *http.Request) (T, error) {
	data := []byte(r.URL.Query().Get("datastar"))
	return ReadFromBytes[T](ns, data)
}

// Dynamically extract a signal from request query params
func ReadFromBytes[T any](ns Signals, data []byte) (out T, err error) {
	if ns.prefix == "" {
		err = json.Unmarshal(data, &out)
		return out, err
	}

	var nested map[string]any
	if err := json.Unmarshal(data, &nested); err != nil {
		return out, fmt.Errorf("signals: unmarshal: %w", err)
	}

	parts := strings.Split(ns.prefix, ".")
	current := nested
	for _, part := range parts {
		val, ok := current[part]
		if !ok {
			return out, fmt.Errorf("signals: missing key %q in payload", part)
		}
		next, ok := val.(map[string]any)
		if !ok {
			return out, fmt.Errorf("signals: key %q is not an object", part)
		}
		current = next
	}

	// Re-marshall un-nested
	currentData, err := json.Marshal(current)
	if err != nil {
		return out, err
	}

	err = json.Unmarshal(currentData, &out)
	return out, err
}

// Get a Signal as expression
func (ns Signals) Ref(field string) Expr {
	return ns.Get(field).Expr
}

// Get a Signal as raw name
//
// Use with: data-bind, data-indicator, data-computed
func (ns Signals) Bind(field string) string {
	return ns.Get(field).Name()
}

// Init returns a data-signals JSON string that initialises this namespace
// with the provided values, correctly nested under the full prefix path.
func (ns Signals) Init(data any) string {
	res, err := ns.marshalData(data)
	if err != nil {
		log.Error("failed to marshall signal data", "error", err)
	}
	return res
}

// Marshal data into object namespaced by signals
func (ns Signals) marshalData(data any) (string, error) {
	encoded, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	if len(ns.prefix) == 0 {
		// TODO?: Check if res is json object
		return string(encoded), nil
	}

	parts := strings.Split(ns.prefix, ".")
	// {"A": {"B": {"C": {"D": DATA }}}}
	var sb strings.Builder
	for _, part := range parts {
		// SECURITY: Only safe if the prefix is known to be safe
		fmt.Fprintf(&sb, `{"%s": `, part)
	}

	// Data payload
	sb.Write(encoded)

	// Close scopes
	for range len(parts) {
		sb.WriteByte('}')
	}

	return sb.String(), nil
}

// exprVal formats a Go value as a JavaScript expression literal for use inside
// a generated expression string.
//
// - Expr/Signal values are inlined as-is (already a safe expression string)
// - Everything else is json marshalled
func exprVal(v any) string {
	switch val := v.(type) {
	case Signal:
		return string(val.Expr)
	case Expr:
		return string(val)
	case string:
		return jsStringLiteral(val)
	default:
		data, err := json.Marshal(val)
		if err != nil {
			log.Error("failed to serialize value", "type", fmt.Sprintf("%T", val))
			return "undefined"
		}
		return string(data)
	}
}

// jsStringLiteral wraps s in single quotes and escapes all characters that
// could break out of a JS string literal context, preventing XSS injection
// when user-controlled values are embedded in Datastar expressions.
//
// Escaped: single quotes, backslashes, newlines, carriage returns, and
// the HTML special characters <, >, & which could interact with script tag parsing.
func jsStringLiteral(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 16)
	b.WriteByte('\'')
	for _, r := range s {
		switch r {
		case '\'':
			b.WriteString(`\'`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '<':
			// Prevent </script> injection if expression is inside a <script> block
			b.WriteString(`\x3C`)
		case '>':
			b.WriteString(`\x3E`)
		case '&':
			b.WriteString(`\x26`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('\'')
	return b.String()
}
