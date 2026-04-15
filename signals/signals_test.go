package signals

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNew_Valid(t *testing.T) {
	tests := []struct {
		parts  []string
		prefix string
	}{
		{[]string{"forms"}, "forms"},
		{[]string{"forms", "billing"}, "forms.billing"},
		{[]string{"forms", "contact_1"}, "forms.contact_1"},
		{[]string{"a", "b", "c"}, "a.b.c"},
		{[]string{"with-dash"}, "with-dash"},
	}
	for _, tt := range tests {
		s := New(tt.parts...)
		if s.prefix != tt.prefix {
			t.Errorf("New(%v).prefix = %q, want %q", tt.parts, s.prefix, tt.prefix)
		}
	}
}

func TestNew_InvalidPanics(t *testing.T) {
	invalid := [][]string{
		{"foo bar"},
		{"foo.bar"},
		{"foo/bar"},
		{""},
		{"foo", "bar!", "baz"},
	}
	for _, parts := range invalid {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("New(%v) should have panicked", parts)
				}
			}()
			New(parts...)
		}()
	}
}

func TestNewFromInput_Valid(t *testing.T) {
	s, err := NewFromInput("forms", "billing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.prefix != "forms.billing" {
		t.Errorf("prefix = %q, want %q", s.prefix, "forms.billing")
	}
}

func TestNewFromInput_Invalid(t *testing.T) {
	_, err := NewFromInput("forms", "bad field")
	if err == nil {
		t.Error("expected error for invalid part")
	}
}

func TestGet_NameAndExpr(t *testing.T) {
	s := New("forms", "contact")
	sig := s.Get("email")

	wantPath := "forms.contact.email"
	wantExpr := "$forms.contact.email"

	if sig.Name() != wantPath {
		t.Errorf("Name() = %q, want %q", sig.Name(), wantPath)
	}
	if sig.String() != wantExpr {
		t.Errorf("String() = %q, want %q", sig.String(), wantExpr)
	}
}

func TestGet_NoPrefix(t *testing.T) {
	s := Signals{}
	sig := s.Get("email")
	if sig.Name() != "email" {
		t.Errorf("Name() = %q, want %q", sig.Name(), "email")
	}
	if sig.String() != "$email" {
		t.Errorf("String() = %q, want %q", sig.String(), "$email")
	}
}

func TestRef(t *testing.T) {
	s := New("forms")
	expr := s.Ref("email")
	if expr.String() != "$forms.email" {
		t.Errorf("Ref() = %q, want %q", expr.String(), "$forms.email")
	}
}

func TestBind(t *testing.T) {
	s := New("forms")
	name := s.Bind("email")
	if name != "forms.email" {
		t.Errorf("Bind() = %q, want %q", name, "forms.email")
	}
}

func TestDerive(t *testing.T) {
	s := New("forms")
	d := s.Derive("table")
	sig := d.Get("order_by")
	if sig.Name() != "forms.table.order_by" {
		t.Errorf("Name() = %q, want %q", sig.Name(), "forms.table.order_by")
	}
}

func TestDerive_EmptyPrefix(t *testing.T) {
	s := Signals{}
	d := s.Derive("forms")
	if d.prefix != "forms" {
		t.Errorf("prefix = %q, want %q", d.prefix, "forms")
	}
}

func TestInit_Nested(t *testing.T) {
	type form struct {
		Email string `json:"email"`
	}
	s := New("forms", "contact")
	result := s.Init(form{Email: "a@b.com"})
	want := `{"forms": {"contact": {"email":"a@b.com"}}}`
	if result != want {
		t.Errorf("Init() = %q, want %q", result, want)
	}
}

func TestInit_NoPrefix(t *testing.T) {
	s := Signals{}
	result := s.Init(map[string]string{"key": "val"})
	want := `{"key":"val"}`
	if result != want {
		t.Errorf("Init() = %q, want %q", result, want)
	}
}

func TestScope_BasicReplacement(t *testing.T) {
	s := New("forms", "table")
	expr := s.Scope(`#s.order_by == 'asc'`)
	want := `$forms.table.order_by == 'asc'`
	if expr.String() != want {
		t.Errorf("Scope() = %q, want %q", expr.String(), want)
	}
}

func TestScope_WithSubstitution(t *testing.T) {
	s := New("forms", "table")
	expr := s.Scope(`#s.order_by == #field`, "field", "products")
	want := `$forms.table.order_by == 'products'`
	if expr.String() != want {
		t.Errorf("Scope() = %q, want %q", expr.String(), want)
	}
}

func TestScope_MultipleSubstitutions(t *testing.T) {
	s := New("ns")
	expr := s.Scope(`#s.x == #a && #s.y == #b`, "a", "foo", "b", "bar")
	want := `$ns.x == 'foo' && $ns.y == 'bar'`
	if expr.String() != want {
		t.Errorf("Scope() = %q, want %q", expr.String(), want)
	}
}

func TestScope_UnknownHashEmittedAsIs(t *testing.T) {
	s := New("ns")
	// #unknown is not in subs, should emit '#' then continue
	expr := s.Scope(`#unknown`)
	if !strings.HasPrefix(expr.String(), "#") {
		t.Errorf("expected unknown # token to be emitted as-is, got %q", expr.String())
	}
}

func TestScope_OddSubstitutionsPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for odd substitutions")
		}
	}()
	s := New("ns")
	s.Scope(`#s.x`, "only_key")
}

func TestScope_NonStringKeyPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for non-string key")
		}
	}()
	s := New("ns")
	s.Scope(`#s.x`, 42, "val")
}

func TestScope_InvalidPlaceholderKeyPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid placeholder key")
		}
	}()
	s := New("ns")
	s.Scope(`#s.x`, "bad key!", "val")
}

func TestExpr_LogicalOps(t *testing.T) {
	e := Expr("$a")
	if e.Not().String() != "(! $a)" {
		t.Errorf("Not() = %q", e.Not().String())
	}
	sig := UnsafeRawExpr("$b")
	if e.And(sig).String() != "($a && $b)" {
		t.Errorf("And(expr) = %q, want %q", e.And(sig).String(), "($a && $b)")
	}
	if e.Or(sig).String() != "($a || $b)" {
		t.Errorf("Or(expr) = %q, want %q", e.Or(sig).String(), "($a || $b)")
	}
}

func TestExpr_ComparisonOps(t *testing.T) {
	e := Expr("$x")
	rhs := UnsafeRawExpr("5")
	tests := []struct {
		got  Expr
		want string
	}{
		{e.Eq(rhs), "($x == 5)"},
		{e.Neq(rhs), "($x != 5)"},
		{e.Gt(rhs), "($x > 5)"},
		{e.GtEq(rhs), "($x >= 5)"},
		{e.Lt(rhs), "($x < 5)"},
		{e.LtEq(rhs), "($x <= 5)"},
	}
	for _, tt := range tests {
		if tt.got.String() != tt.want {
			t.Errorf("got %q, want %q", tt.got.String(), tt.want)
		}
	}
}

func TestExpr_ArithmeticOps(t *testing.T) {
	e := Expr("$x")
	rhs := UnsafeRawExpr("2")
	tests := []struct {
		got  Expr
		want string
	}{
		{e.Add(rhs), "($x + 2)"},
		{e.Sub(rhs), "($x - 2)"},
		{e.Mul(rhs), "($x * 2)"},
		{e.Div(rhs), "($x / 2)"},
		{e.Mod(rhs), "($x % 2)"},
	}
	for _, tt := range tests {
		if tt.got.String() != tt.want {
			t.Errorf("got %q, want %q", tt.got.String(), tt.want)
		}
	}
}

func TestExpr_IfThen(t *testing.T) {
	e := Expr("$loading")
	got := e.IfThen("Saving...", "Submit")
	want := `($loading ? 'Saving...' : 'Submit')`
	if got.String() != want {
		t.Errorf("IfThen() = %q, want %q", got.String(), want)
	}
}

func TestExpr_Toggle(t *testing.T) {
	e := Expr("$open")
	got := e.Toggle()
	want := "($open = (! $open))"
	if got.String() != want {
		t.Errorf("Toggle() = %q, want %q", got.String(), want)
	}
}

func TestExpr_EmptyNotEmpty(t *testing.T) {
	e := Expr("$errMsg")
	if e.Empty().String() != "($errMsg == '')" {
		t.Errorf("Empty() = %q", e.Empty().String())
	}
	if e.NotEmpty().String() != "($errMsg != '')" {
		t.Errorf("NotEmpty() = %q", e.NotEmpty().String())
	}
}

func TestExpr_BackendActions(t *testing.T) {
	e := Expr("$x")
	tests := []struct {
		got  Expr
		want string
	}{
		{e.Post("/api/foo"), "@post('/api/foo')"},
		{e.Get("/api/foo"), "@get('/api/foo')"},
		{e.Put("/api/foo"), "@put('/api/foo')"},
		{e.Patch("/api/foo"), "@patch('/api/foo')"},
		{e.Delete("/api/foo"), "@delete('/api/foo')"},
		{DataPost("/api/foo"), "@post('/api/foo')"},
		{DataGet("/api/foo"), "@get('/api/foo')"},
		{DataPut("/api/foo"), "@put('/api/foo')"},
		{DataPatch("/api/foo"), "@patch('/api/foo')"},
		{DataDelete("/api/foo"), "@delete('/api/foo')"},
	}
	for _, tt := range tests {
		if tt.got.String() != tt.want {
			t.Errorf("got %q, want %q", tt.got.String(), tt.want)
		}
	}
}

// --- jsStringLiteral / exprVal ---

func TestJsStringLiteral_Escaping(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", `'hello'`},
		{"it's", `'it\'s'`},
		{`back\slash`, `'back\\slash'`},
		{"new\nline", `'new\nline'`},
		{"cr\r", `'cr\r'`},
		{"<script>", `'\x3Cscript\x3E'`},
		{"a&b", `'a\x26b'`},
	}
	for _, tt := range tests {
		got := jsStringLiteral(tt.input)
		if got != tt.want {
			t.Errorf("jsStringLiteral(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExprVal_Types(t *testing.T) {
	sig := New("ns").Get("field")
	// Signal → inlined as-is
	if exprVal(sig) != "$ns.field" {
		t.Errorf("exprVal(Signal) = %q", exprVal(sig))
	}
	// Expr → inlined
	e := Expr("$x")
	if exprVal(e) != "$x" {
		t.Errorf("exprVal(Expr) = %q", exprVal(e))
	}
	// string → quoted
	if exprVal("hello") != `'hello'` {
		t.Errorf("exprVal(string) = %q", exprVal("hello"))
	}
	// int → JSON
	if exprVal(42) != "42" {
		t.Errorf("exprVal(int) = %q", exprVal(42))
	}
	// bool → JSON
	if exprVal(true) != "true" {
		t.Errorf("exprVal(bool) = %q", exprVal(true))
	}
}

// --- ReadFromBody ---

func TestReadFromBody(t *testing.T) {
	type Form struct {
		Email string `json:"email"`
	}
	s := New("forms")
	body := `{"forms":{"email":"test@example.com"}}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	form, err := ReadFromBody[Form](s, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if form.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q", form.Email, "test@example.com")
	}
}

func TestReadFromBody_MissingKey(t *testing.T) {
	type Form struct{ Email string }
	s := New("forms", "contact")
	body := `{"forms":{"email":"x"}}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	_, err := ReadFromBody[Form](s, req)
	if err == nil {
		t.Error("expected error for missing nested key")
	}
}

func TestReadFromQuery(t *testing.T) {
	type Form struct {
		Name string `json:"name"`
	}
	s := New("data")
	payload := `{"data":{"name":"Alice"}}`
	req := httptest.NewRequest(http.MethodGet, "/?datastar="+payload, nil)
	form, err := ReadFromQuery[Form](s, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if form.Name != "Alice" {
		t.Errorf("Name = %q, want %q", form.Name, "Alice")
	}
}

func TestRead_UnsupportedMethod(t *testing.T) {
	s := New("ns")
	req := httptest.NewRequest(http.MethodHead, "/", nil)
	_, err := Read[map[string]any](s, req)
	if err == nil {
		t.Error("expected error for unsupported method")
	}
}
