package dexpr

import (
	"errors"
	"fmt"
	"github.com/lawrencewoodman/dlit"
	"go/token"
	"math"
	"testing"
)

func TestMustNew(t *testing.T) {
	cases := []struct {
		in      string
		wantStr string
	}{
		{"a+b", "a+b"},
		{"income", "income"},
		{"6.6", "6.6"},
	}
	funcs := map[string]CallFun{}

	for _, c := range cases {
		got := MustNew(c.in, funcs)
		if got.String() != c.wantStr {
			t.Errorf("MustNew(%v) - got: %s, want: %s", c.in, got, c.wantStr)
		}
	}
}

func TestMustNew_panic(t *testing.T) {
	expr := "/bob harry"
	funcs := map[string]CallFun{}
	wantPanic := InvalidExprError{"/bob harry", ErrSyntax}
	paniced := false
	defer func() {
		if r := recover(); r != nil {
			if r.(string) == wantPanic.Error() {
				paniced = true
			} else {
				t.Errorf("MustNew(%s) - got panic: %s, wanted: %s", expr, r, wantPanic)
			}
		}
	}()
	MustNew(expr, funcs)
	if !paniced {
		t.Errorf("MustNew(%s) - failed to panic with: %s", expr, wantPanic)
	}
}

func TestNew_errors(t *testing.T) {
	cases := []struct {
		in        string
		wantError error
	}{
		{"7 {} 3", InvalidExprError{"7 {} 3", ErrSyntax}},
		{"8/cot££t", InvalidExprError{"8/cot££t", ErrSyntax}},
		{"[lit{fred", InvalidExprError{"[lit{fred", ErrSyntax}},
		{"[lit{fred}", InvalidExprError{"[lit{fred}", ErrSyntax}},
		{"[]lit{fred", InvalidExprError{"[]lit{fred", ErrSyntax}},
		{"func() bool {return 1==1}",
			InvalidExprError{"func() bool {return 1==1}", ErrSyntax},
		},
		{"10 & 101", InvalidExprError{"10 & 101", InvalidOpError(token.AND)}},

		/* Composite literals */
		{"[]lit{7,9,2}[3] == 9",
			InvalidExprError{"[]lit{7,9,2}[3] == 9", ErrInvalidIndex},
		},
		{"[]int{7,9,2}[1] == 9",
			InvalidExprError{
				"[]int{7,9,2}[1] == 9",
				ErrInvalidCompositeType,
			},
		},
		{"[]string{\"fred\",\"bob\",\"alf\"}[1] == \"bob\"",
			InvalidExprError{
				"[]string{\"fred\",\"bob\",\"alf\"}[1] == \"bob\"",
				ErrInvalidCompositeType,
			},
		},

		/* Indexing non indexable values */
		{"7[0] == 4",
			InvalidExprError{
				"7[0] == 4",
				ErrTypeNotIndexable,
			}},
		{"7.2[0] == 4",
			InvalidExprError{
				"7.2[0] == 4",
				ErrTypeNotIndexable,
			}},

		/* map not implemented */
		{"map[lit]lit{\"fred\": 7, \"bob\": 9, \"alf\": 2}[\"bob\"] == 8",
			InvalidExprError{
				"map[lit]lit{\"fred\": 7, \"bob\": 9, \"alf\": 2}[\"bob\"] == 8",
				ErrSyntax,
			}},
		{"map[lit]lit{\"fred\": 7, \"bob\": 9, \"alf\": 2}[\"bob\"] == 9",
			InvalidExprError{
				"map[lit]lit{\"fred\": 7, \"bob\": 9, \"alf\": 2}[\"bob\"] == 9",
				ErrSyntax,
			}},
	}
	funcs := map[string]CallFun{}
	for _, c := range cases {
		_, err := New(c.in, funcs)
		if err == nil {
			t.Errorf("New(%s) no error, wanted: %s", c.in, err)
		}
		if err != c.wantError {
			t.Errorf("New(%s) got error: %s, wanted: %s", c.in, err, c.wantError)
		}
	}
}

func TestDexprEval(t *testing.T) {
	expr := "roundto(bob, 2) + 7"
	want := dlit.MustNew(26.12)
	funcs := map[string]CallFun{"roundto": roundTo}
	vars := map[string]*dlit.Literal{"bob": dlit.MustNew(19.1234)}
	got := Eval(expr, funcs, vars)
	if got.String() != want.String() {
		t.Errorf("Eval: got: %s, want: %s", got, want)
	}
}

func TestDexprEvalBool(t *testing.T) {
	cases := []struct {
		expr string
		want bool
	}{
		{expr: "roundto(bob, 2) + 7 == 26.12", want: true},
		{expr: "roundto(bob, 2) + 7 == 26.11", want: false},
	}
	for _, c := range cases {
		funcs := map[string]CallFun{"roundto": roundTo}
		vars := map[string]*dlit.Literal{"bob": dlit.MustNew(19.1234)}
		got, err := EvalBool(c.expr, funcs, vars)
		if got != c.want {
			t.Errorf("Eval: got: %t, want: %t", got, c.want)
		}
		if err != nil {
			t.Errorf("Eval: err: %s", err)
		}
	}
}

func TestDexprEval_errors(t *testing.T) {
	expr := "7 {} 3"
	wantErr := InvalidExprError{"7 {} 3", ErrSyntax}
	funcs := map[string]CallFun{}
	vars := map[string]*dlit.Literal{}
	got := Eval(expr, funcs, vars)
	if err := got.Err(); err == nil || err.Error() != wantErr.Error() {
		t.Errorf("Eval: gotErr: %s, want: %s", err, wantErr)
	}
}

func TestDexprEvalBool_errors(t *testing.T) {
	expr := "7 {} 3"
	want := false
	wantErr := InvalidExprError{"7 {} 3", ErrSyntax}
	funcs := map[string]CallFun{}
	vars := map[string]*dlit.Literal{}
	got, err := EvalBool(expr, funcs, vars)
	if got != want {
		t.Errorf("Eval: got: %t, want: %t", got, want)
	}
	if err == nil || err.Error() != wantErr.Error() {
		t.Errorf("Eval: gotErr: %s, want: %s", err, wantErr)
	}
}

func TestEval_noerrors(t *testing.T) {
	cases := []struct {
		in   string
		want *dlit.Literal
	}{
		{"1 == 1", dlit.MustNew(true)},
		{"1 == 2", dlit.MustNew(false)},
		{"2.6 + 2.5", dlit.MustNew(5.1)},
		{"-2 + -2", dlit.MustNew(-4)},
		{"-2.5 + -2.6", dlit.MustNew(-5.1)},
		{"-2 - 3", dlit.MustNew(-5)},
		{"-2.5 - 3.6", dlit.MustNew(-6.1)},
		{"8 - 9", dlit.MustNew(-1)},
		{"a + numStrB", dlit.MustNew(7)},
		{"8/4", dlit.MustNew(2)},
		{"1/4", dlit.MustNew(0.25)},
		{"8*4", dlit.MustNew(32)},
		{fmt.Sprintf("%d * 1", int64(math.MinInt64)),
			dlit.MustNew(int64(math.MinInt64))},
		{fmt.Sprintf("%d * 1", int64(math.MaxInt64)),
			dlit.MustNew(int64(math.MaxInt64))},
		{fmt.Sprintf("(%d / 2) * 2", int64(math.MinInt64)),
			dlit.MustNew(int64(math.MinInt64))},
		{fmt.Sprintf("((%d+-1) / 2) * 2", int64(math.MaxInt64)),
			dlit.MustNew(int64(math.MaxInt64) - 1)},

		/* Tests that unary negation works properly */
		{fmt.Sprintf("%d + 0", int64(math.MinInt64)),
			dlit.MustNew(int64(math.MinInt64))},

		{"roundto(5.567, 2)", dlit.MustNew(5.57)},
		{"roundto(-17.5, 0)", dlit.MustNew(-17)},

		/* Tests that integer overlow is prevented by switching to Float */
		{fmt.Sprintf("%d+%d", int64(math.MaxInt64), int64(math.MaxInt64)),
			dlit.MustNew(float64(math.MaxInt64) + float64(math.MaxInt64))},
		{fmt.Sprintf("%d*%d", int64(math.MaxInt64), int64(math.MaxInt64)),
			dlit.MustNew(float64(math.MaxInt64) * float64(math.MaxInt64))},
		{fmt.Sprintf("%d*%d", int64(math.MinInt64), int64(math.MinInt64)),
			dlit.MustNew(float64(math.MinInt64) * float64(math.MinInt64))},
		{fmt.Sprintf("%d*%d", int64(math.MaxInt64), int64(math.MinInt64)),
			dlit.MustNew(float64(math.MaxInt64) * float64(math.MinInt64))},
		{fmt.Sprintf("%d+1", int64(math.MaxInt64)),
			dlit.MustNew(float64(math.MaxInt64) + 1)},
		{fmt.Sprintf("%d-1", int64(math.MinInt64)),
			dlit.MustNew(float64(math.MinInt64) - 1)},
		{fmt.Sprintf("%d + -1", int64(math.MinInt64)),
			dlit.MustNew(float64(math.MinInt64) + -1)},
		{fmt.Sprintf("%d - %d", int64(math.MaxInt64), int64(math.MinInt64)),
			dlit.MustNew(float64(math.MaxInt64) - float64(math.MinInt64))},
		{fmt.Sprintf("%d - -1", int64(math.MaxInt64)),
			dlit.MustNew(float64(math.MaxInt64) - -1)},
		{fmt.Sprintf("%d - 1", int64(math.MinInt64)),
			dlit.MustNew(float64(math.MinInt64) - -1)},
		{fmt.Sprintf("%d*2", int64(math.MaxInt64)),
			dlit.MustNew(float64(math.MaxInt64) * 2)},
		{fmt.Sprintf("%d*2", int64(math.MinInt64)),
			dlit.MustNew(float64(math.MinInt64) * 2)},
		{fmt.Sprintf("%d/0.5", int64(math.MaxInt64)),
			dlit.MustNew(float64(math.MaxInt64) / 0.5)},
		{fmt.Sprintf("%d/0.5", int64(math.MinInt64)),
			dlit.MustNew(float64(math.MinInt64) / 0.5)},

		/* Test the Floats work properly */
		{fmt.Sprintf("%f+0", float64(math.MaxFloat64)),
			dlit.MustNew(float64(math.MaxFloat64))},
		{fmt.Sprintf("0+%f", float64(math.MaxFloat64)),
			dlit.MustNew(float64(math.MaxFloat64))},
		{fmt.Sprintf("%0.324f+%0.324f",
			float64(math.MaxFloat64)-float64(math.MaxFloat64)/4,
			float64(math.MaxFloat64)/4),
			dlit.MustNew(float64(math.MaxFloat64)),
		},
		{fmt.Sprintf("%0.324f+%0.324f",
			float64(math.MaxFloat64)/4,
			float64(math.MaxFloat64)-float64(math.MaxFloat64)/4),
			dlit.MustNew(float64(math.MaxFloat64)),
		},
		{fmt.Sprintf("%f-%0.324f", float64(math.MaxFloat64),
			float64(math.SmallestNonzeroFloat64)),
			dlit.MustNew(
				float64(math.MaxFloat64) - float64(math.SmallestNonzeroFloat64),
			),
		},
		{fmt.Sprintf("%0.324f*2", float64(math.SmallestNonzeroFloat64)),
			dlit.MustNew(float64(math.SmallestNonzeroFloat64) * 2),
		},
		{fmt.Sprintf("%0.324f*%0.324f", float64(math.SmallestNonzeroFloat64),
			float64(math.SmallestNonzeroFloat64)),
			dlit.MustNew(0),
		},
		{fmt.Sprintf("%f*%0.324f", float64(math.MaxFloat64),
			float64(math.SmallestNonzeroFloat64)),
			dlit.MustNew(float64(math.MaxFloat64) *
				float64(math.SmallestNonzeroFloat64)),
		},
		{fmt.Sprintf("%0.324f/2", float64(math.SmallestNonzeroFloat64)),
			dlit.MustNew(0),
		},
		{fmt.Sprintf("%0.324f/4", float64(math.MaxFloat64)),
			dlit.MustNew(float64(math.MaxFloat64) / 4),
		},

		/* Check operator precedence */
		{"5 * 2 + 3", dlit.MustNew(13)},
		{"3 + 5 * 2", dlit.MustNew(13)},
		{"3 + (5 * 2)", dlit.MustNew(13)},
		{"(3 + 5) * 2", dlit.MustNew(16)},
	}
	vars := map[string]*dlit.Literal{
		"a":       dlit.MustNew(4),
		"numStrB": dlit.MustNew("3"),
	}
	funcs := map[string]CallFun{
		"roundto": roundTo,
	}
	for _, c := range cases {
		dexpr, err := New(c.in, funcs)
		if err != nil {
			t.Errorf("New(%s) err: %s", c.in, err)
		}
		got := dexpr.Eval(vars)
		if err := got.Err(); err != nil {
			t.Errorf("Eval(vars) in: %s, err: %s", c.in, err)
		}

		if got.String() != c.want.String() {
			t.Errorf("Eval(vars) in: %s, got: %s, want: %s", c.in, got, c.want)
		}
	}
}

func TestEval_errors(t *testing.T) {
	cases := []struct {
		in   string
		want *dlit.Literal
	}{
		{"8/bob", dlit.MustNew(
			InvalidExprError{"8/bob", VarNotExistError("bob")}),
		},
		{"8/(1 == 1)", dlit.MustNew(
			InvalidExprError{"8/(1 == 1)", ErrIncompatibleTypes},
		)},
		{"8/0", dlit.MustNew(InvalidExprError{"8/0", ErrDivByZero})},
		{"bob(5.567, 2)", dlit.MustNew(
			InvalidExprError{"bob(5.567, 2)", FunctionNotExistError("bob")},
		)},
		{"roundto(5.567, 2, 9, 23)", dlit.MustNew(
			InvalidExprError{"roundto(5.567, 2, 9, 23)",
				FunctionError{"roundto", errTooManyArguments}},
		)},

		{"[]lit{numStrA, numStrB, numStrC}[2] == 3", dlit.MustNew(
			InvalidExprError{
				"[]lit{numStrA, numStrB, numStrC}[2] == 3",
				VarNotExistError("numStrC")},
		)},

		{fmt.Sprintf("%f+%f", float64(math.MaxFloat64), float64(math.MaxFloat64)),
			dlit.MustNew(InvalidExprError{
				fmt.Sprintf("%f+%f", float64(math.MaxFloat64), float64(math.MaxFloat64)),
				ErrUnderflowOverflow,
			}),
		},
		{fmt.Sprintf("%f*%f", float64(math.MaxFloat64), float64(math.MaxFloat64)),
			dlit.MustNew(InvalidExprError{
				fmt.Sprintf("%f*%f", float64(math.MaxFloat64), float64(math.MaxFloat64)),
				ErrUnderflowOverflow,
			}),
		},
		{fmt.Sprintf("%0.324f+%0.324f",
			float64(math.MaxFloat64),
			float64(math.MaxFloat64)/4,
		),
			dlit.MustNew(InvalidExprError{
				fmt.Sprintf("%0.324f+%0.324f",
					float64(math.MaxFloat64),
					float64(math.MaxFloat64)/4,
				),
				ErrUnderflowOverflow,
			}),
		},
		{fmt.Sprintf("-%0.324f-%0.324f",
			float64(math.MaxFloat64),
			float64(math.MaxFloat64)/4,
		),
			dlit.MustNew(InvalidExprError{
				fmt.Sprintf("-%0.324f-%0.324f",
					float64(math.MaxFloat64),
					float64(math.MaxFloat64)/4,
				),
				ErrUnderflowOverflow,
			}),
		},
		{fmt.Sprintf("-%0.324f + -%0.324f",
			float64(math.MaxFloat64),
			float64(math.MaxFloat64)/4,
		),
			dlit.MustNew(InvalidExprError{
				fmt.Sprintf("-%0.324f + -%0.324f",
					float64(math.MaxFloat64),
					float64(math.MaxFloat64)/4,
				),
				ErrUnderflowOverflow,
			}),
		},
		{fmt.Sprintf("%0.324f - -%0.324f",
			float64(math.MaxFloat64),
			float64(math.MaxFloat64)/4,
		),
			dlit.MustNew(InvalidExprError{
				fmt.Sprintf("%0.324f - -%0.324f",
					float64(math.MaxFloat64),
					float64(math.MaxFloat64)/4,
				),
				ErrUnderflowOverflow,
			}),
		},
		{fmt.Sprintf("%0.324f*2", float64(math.MaxFloat64)),
			dlit.MustNew(InvalidExprError{
				fmt.Sprintf("%0.324f*2", float64(math.MaxFloat64)),
				ErrUnderflowOverflow,
			}),
		},
		{fmt.Sprintf("%0.324f * -2", float64(math.MaxFloat64)),
			dlit.MustNew(InvalidExprError{
				fmt.Sprintf("%0.324f * -2", float64(math.MaxFloat64)),
				ErrUnderflowOverflow,
			}),
		},
		{fmt.Sprintf("%0.324f / 0.5", float64(math.MaxFloat64)),
			dlit.MustNew(InvalidExprError{
				fmt.Sprintf("%0.324f / 0.5", float64(math.MaxFloat64)),
				ErrUnderflowOverflow,
			}),
		},
		{fmt.Sprintf("%0.324f / -0.5", float64(math.MaxFloat64)),
			dlit.MustNew(InvalidExprError{
				fmt.Sprintf("%0.324f / -0.5", float64(math.MaxFloat64)),
				ErrUnderflowOverflow,
			}),
		},
	}
	vars := map[string]*dlit.Literal{
		"a":       dlit.MustNew(4),
		"numStrB": dlit.MustNew("3"),
	}
	funcs := map[string]CallFun{
		"roundto": roundTo,
	}
	for _, c := range cases {
		dexpr, err := New(c.in, funcs)
		if err != nil {
			t.Fatalf("New(%s) err: %s", c.in, err)
		}
		got := dexpr.Eval(vars)
		gotErr := got.Err()
		wantErr := c.want.Err()
		if gotErr != wantErr {
			t.Errorf("Eval(vars) in: %s, got: %s, want: %s", c.in, got, c.want)
		}
	}
}

func TestEvalBool_noErrors(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"1 == 1", true},
		{"1 == 2", false},
		{"2.5 == 2.5", true},
		{"2.5 == 3.5", false},
		{"1 == 1.5", false},

		/* Becareful of float == int comparison */
		{"1.0 == 1", true},
		{"1 == 1.0", true},

		{"numStrB == 3", true},
		{"numStrB == 3.0", true},
		{"3 == numStrB", true},
		{"3.0 == numStrB", true},
		{"a == 4", true},
		{"a == 5", false},
		{"a == a", true},
		{"a == b", false},
		{"\"hello\" == \"hello\"", true},
		{"\"hllo\" == \"hello\"", false},
		{"\"hllo\" == 7", false},
		{"str == \"hello\"", true},
		{"str == \"helo\"", false},
		{"numStrA == 3", false},
		{"numStrA == 4", true},
		{"numStrA == numStrA", true},
		{"numStrA == numStrB", false},
		{"numStrC == numStrD", false},

		/* Check that keyword tokens are parsed as variables */
		{"break == 1", true},
		{"break == 2", false},
		{"case == 2", true},
		{"case == 3", false},
		{"chan == 3", true},
		{"chan == 4", false},
		{"const == 4", true},
		{"const == 5", false},
		{"continue == 5", true},
		{"continue == 6", false},
		{"default == 6", true},
		{"default == 7", false},
		{"defer == 7", true},
		{"defer == 8", false},
		{"else == 8", true},
		{"else == 9", false},
		{"fallthrough == 9", true},
		{"fallthrough == 10", false},
		{"for == 10", true},
		{"for == 11", false},
		{"func == 11", true},
		{"func == 12", false},
		{"go == 12", true},
		{"go == 13", false},
		{"goto == 13", true},
		{"goto == 14", false},
		{"if == 14", true},
		{"if == 15", false},
		{"import == 15", true},
		{"import == 16", false},
		{"interface == 16", true},
		{"interface == 17", false},
		{"map == 17", true},
		{"map == 18", false},
		{"package == 18", true},
		{"package == 19", false},
		{"range == 19", true},
		{"range == 20", false},
		{"return == 20", true},
		{"return == 21", false},
		{"select == 21", true},
		{"select == 22", false},
		{"struct == 22", true},
		{"struct == 23", false},
		{"switch == 23", true},
		{"switch == 24", false},
		{"type == 24", true},
		{"type == 25", false},
		{"var == 25", true},
		{"var == 26", false},

		{"a != 4", false},
		{"a != 5", true},
		{"a != a", false},
		{"a != b", true},
		{"\"hello\" != \"hello\"", false},
		{"\"hllo\" != \"hello\"", true},
		{"\"hllo\" != 7", true},
		{"str != \"hello\"", false},
		{"str != \"helo\"", true},
		{"numStrA != 3", true},
		{"numStrA != 4", false},
		{"numStrA != numStrA", false},
		{"numStrA != numStrB", true},
		{"numStrC != numStrD", true},

		/* Ensure that bools are not used for comparison */
		{"\"true\" == 1", false},
		{"\"true\" == 1.0", false},
		{"\"true\" == \"TRUE\"", false},
		{"\"TRUE\" == \"TRUE\"", true},
		{"\"false\" == \"FALSE\"", false},
		{"\"FALSE\" == \"FALSE\"", true},
		{"\"false\" ==  0", false},
		{"\"false\" ==  0.0", false},
		{"\"true\" != 0", true},
		{"\"true\" != 1.0", true},
		{"\"true\" != \"TRUE\"", true},
		{"\"TRUE\" != \"TRUE\"", false},
		{"\"false\" != \"FALSE\"", true},
		{"\"FALSE\" != \"FALSE\"", false},
		{"\"false\" !=  0", true},
		{"\"false\" !=  0.0", true},

		{"6 < 7", true},
		{"7 < 7", false},
		{"8 < 7", false},
		{"6.7 < 7", true},
		{"6.7 < 7.7", true},
		{"7 < 7.2", true},
		{"7 < 6.7", false},
		{"3 < a", true},
		{"4 < a", false},
		{"a < 5", true},
		{"a < 4", false},
		{"b < a", true},
		{"b < b", false},
		{"a < b", false},
		{"3 < numStrA", true},
		{"4 < numStrA", false},
		{"numStrA < 5", true},
		{"numStrA < 4", false},
		{"numStrB < numStrA", true},
		{"numStrB < numStrB", false},
		{"numStrA < numStrB", false},
		{"numStrA < numStrC", true},
		{"numStrD < numStrC", true},
		{"6 <= 7", true},
		{"7 <= 7", true},
		{"8 <= 7", false},
		{"6.7 <= 7", true},
		{"6.7 <= 7.7", true},
		{"7 <= 7.2", true},
		{"7 <= 6.7", false},
		{"b <= a", true},
		{"a <= a", true},
		{"a <= b", false},
		{"3 <= numStrA", true},
		{"4 <= numStrA", true},
		{"5 <= numStrA", false},
		{"5.5 <= numStrA", false},
		{"numStrA <= 5", true},
		{"numStrA <= 4", true},
		{"numStrA <= 3", false},
		{"numStrB <= numStrA", true},
		{"numStrB <= numStrB", true},
		{"numStrA <= numStrB", false},
		{"numStrA <= numStrC", true},
		{"numStrD <= numStrC", true},
		{"6 > 7", false},
		{"7 > 7", false},
		{"8 > 7", true},
		{"6.7 > 7", false},
		{"6.7 > 7.7", false},
		{"7 > 7.2", false},
		{"b > a", false},
		{"a > b", true},
		{"3 > numStrA", false},
		{"4 > numStrA", false},
		{"5 > numStrA", true},
		{"5.5 > numStrA", true},
		{"numStrA > 5", false},
		{"numStrA > 4", false},
		{"numStrA > 3", true},
		{"numStrB > numStrA", false},
		{"numStrB > numStrB", false},
		{"numStrA > numStrB", true},
		{"numStrA > numStrC", false},
		{"numStrD > numStrC", false},
		{"6 >= 7", false},
		{"7 >= 7", true},
		{"8 >= 7", true},
		{"6.7 >= 7", false},
		{"6.7 >= 7.7", false},
		{"7.2 >= 7", true},
		{"7.2 >= 7.2", true},
		{"b >= a", false},
		{"a >= b", true},
		{"3 >= numStrA", false},
		{"4 >= numStrA", true},
		{"5 >= numStrA", true},
		{"5.5 >= numStrA", true},
		{"numStrA >= 5", false},
		{"numStrA >= 4", true},
		{"numStrA >= 3", true},
		{"numStrB >= numStrA", false},
		{"numStrB >= numStrB", true},
		{"numStrA >= numStrB", true},
		{"numStrA >= numStrC", false},
		{"numStrD >= numStrC", false},
		{"5 + 1.5 > 6", true},
		{"5 + 1 > 6", false},
		{"a + b > 6", true},
		{"a + b > 7", false},
		{"a + b > 8", false},
		{"numStrA + numStrB > 6", true},
		{"numStrA + numStrB > 7", false},
		{"numStrA + numStrB > 8", false},
		{"numStrC + numStrD > 7", true},
		{"numStrC + numStrD == 8.0", true},
		{"numStrC + numStrD == 8", true},
		{"numStrC + numStrD > 8", false},

		{"trueStr", true},
		{"!trueStr", false},

		{"9 > 8 && 2 < 3", true},
		{"9 > 9 && 2 < 3", false},
		{"9 > 8 && 3 < 3", false},
		{"9 > 9 && 3 < 3", false},
		{"9 > 8 && 2 < 3 && 7 > 2", true},
		{"9 > 8 && 2 < 3 && 7 > 7", false},
		{"9 > 8 || 2 < 3", true},
		{"9 > 9 || 2 < 3", true},
		{"9 > 8 || 3 < 3", true},
		{"9 > 9 || 3 < 3", false},
		{"9 > 8 || 2 < 3 || 7 > 2", true},
		{"8 > 8 || 3 < 3 || 7 > 7", false},
		{"9 + (8 + 2) > 18", true},
		{"9 + (8 + 2) > 19", false},
		{"roundto(8+2.25, 1) == 10.3", true},
		{"roundto(8+2.25, 1) == 10.25", false},

		/* Check that chars can be used and strings indexed */
		{"\"Hello world\"[6] == 'w'", true},
		{"\"Hello world\"[6] == 'h'", false},
		{"\"Hello world\"[6] == \"w\"", true},
		{"\"Hello world\"[6] == \"h\"", false},

		/* Check composite literals can be used */
		{"[]lit{7,9,2}[1] == 9", true},
		{"[]lit{7,9,2}[1] == 8", false},
		{"[3]lit{7,9,2}[1] == 9", true},
		{"[3]lit{7,9,2}[1] == 8", false},
		{"[]lit{7.8,9.4,2.3}[1] == 9.4", true},
		{"[]lit{7.8,9.4,2.3}[1] == 7.8", false},
		{"[]lit{7.8,9.4,2.3}[1] == []lit{12.8,9.4,9.3}[1]", true},
		{"[]lit{7.8,9.4,2.3}[1] == []lit{12.8,9.4,9.3}[2]", false},
		{"[]lit{numStrA, numStrB, numStrC}[2] == 4.5", true},
		{"[]lit{numStrA, numStrB, numStrC}[2] == 3", false},
		{"[]lit{numStrA, numStrB, numStrC}[2] == []lit{numStrA, numStrB, numStrE}[2]",
			true},
		{"[]lit{numStrA, numStrB, numStrC}[2] == []lit{numStrA, numStrB, numStrE}[1]",
			false},
		{"[]lit{\"fred\", \"bob\", \"alf\"}[2] == \"alf\"", true},
		{"[]lit{\"fred\", \"bob\", \"alf\"}[2] == \"bob\"", false},
		{"[3]lit{\"fred\", \"bob\", \"alf\"}[2] == \"alf\"", true},
		{"[3]lit{\"fred\", \"bob\", \"alf\"}[2] == \"bob\"", false},

		/*
			{"isFrom(5)", true},
			{"isFrom(true)", true},
		*/
	}
	vars := map[string]*dlit.Literal{
		"a":           dlit.MustNew(4),
		"b":           dlit.MustNew(3),
		"c":           dlit.MustNew(4.5),
		"d":           dlit.MustNew(3.5),
		"str":         dlit.MustNew("hello"),
		"numStrA":     dlit.MustNew("4"),
		"numStrB":     dlit.MustNew("3"),
		"numStrC":     dlit.MustNew("4.5"),
		"numStrD":     dlit.MustNew("3.5"),
		"numStrE":     dlit.MustNew("4.5"),
		"trueStr":     dlit.MustNew(true),
		"break":       dlit.MustNew(1),
		"case":        dlit.MustNew(2),
		"chan":        dlit.MustNew(3),
		"const":       dlit.MustNew(4),
		"continue":    dlit.MustNew(5),
		"default":     dlit.MustNew(6),
		"defer":       dlit.MustNew(7),
		"else":        dlit.MustNew(8),
		"fallthrough": dlit.MustNew(9),
		"for":         dlit.MustNew(10),
		"func":        dlit.MustNew(11),
		"go":          dlit.MustNew(12),
		"goto":        dlit.MustNew(13),
		"if":          dlit.MustNew(14),
		"import":      dlit.MustNew(15),
		"interface":   dlit.MustNew(16),
		"map":         dlit.MustNew(17),
		"package":     dlit.MustNew(18),
		"range":       dlit.MustNew(19),
		"return":      dlit.MustNew(20),
		"select":      dlit.MustNew(21),
		"struct":      dlit.MustNew(22),
		"switch":      dlit.MustNew(23),
		"type":        dlit.MustNew(24),
		"var":         dlit.MustNew(25),
	}
	funcs := map[string]CallFun{
		"roundto": roundTo,
	}
	for _, c := range cases {
		dexpr, err := New(c.in, funcs)
		if err != nil {
			t.Fatalf("New(%s) err: %s", c.in, err)
		}
		got, err := dexpr.EvalBool(vars)
		if err != nil {
			t.Errorf("EvalBool(vars, %v) err == %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("EvalBool(vars, %v) == %v, want %v", c.in, got, c.want)
		}
	}
}

func TestString(t *testing.T) {
	cases := []string{
		"1 == 1",
		"2.5 == 2.5",
		"1.0 == 1",
		"numStr == 3",
		"\"true\" == \"TRUE\"",
		"5 + 1.5 > 6",
	}
	funcs := map[string]CallFun{}
	for _, c := range cases {
		dexpr, err := New(c, funcs)
		if err != nil {
			t.Errorf("New(%s) err: %s", c, err)
		}
		got := dexpr.String()
		if got != c {
			t.Errorf("String() got %s, want: %s", got, c)
		}
	}
}

func TestEvalBool_errors(t *testing.T) {
	vars := map[string]*dlit.Literal{
		"anError": dlit.MustNew(errors.New("this is an error")),
	}
	cases := []struct {
		in        string
		want      bool
		wantError error
	}{
		{"7 + 8", false, InvalidExprError{"7 + 8", ErrIncompatibleTypes}},
		{"7 < \"hello\"", false,
			InvalidExprError{"7 < \"hello\"", ErrIncompatibleTypes},
		},
		{"\"world\" > 2.1", false,
			InvalidExprError{"\"world\" > 2.1", ErrIncompatibleTypes},
		},
		{"7 && 9", false, InvalidExprError{"7 && 9", ErrIncompatibleTypes}},
		{"total > 20",
			false,
			InvalidExprError{"total > 20", VarNotExistError("total")},
		},
		{"!bob",
			false,
			InvalidExprError{"!bob", VarNotExistError("bob")},
		},
		{"7 + -bob > 2",
			false,
			InvalidExprError{"7 + -bob > 2", VarNotExistError("bob")},
		},
		{"20 < total",
			false,
			InvalidExprError{"20 < total", VarNotExistError("total")},
		},
		{"bob(8+2.257) == 7", false,
			InvalidExprError{"bob(8+2.257) == 7", FunctionNotExistError("bob")},
		},
		{"-\"something\"", false,
			InvalidExprError{"-\"something\"", ErrIncompatibleTypes},
		},
		{"!5.2", false, InvalidExprError{"!5.2", ErrIncompatibleTypes}},
		{"anError == anError",
			false,
			InvalidExprError{"anError == anError", vars["anError"].Err()},
		},
		{"anError != anError",
			false,
			InvalidExprError{"anError != anError", vars["anError"].Err()},
		},
	}
	funcs := map[string]CallFun{}
	for _, c := range cases {
		dexpr, err := New(c.in, funcs)
		if err != nil {
			t.Fatalf("New(%s) err: %s", c.in, err)
		}
		got, err := dexpr.EvalBool(vars)
		if got != c.want {
			t.Errorf("EvalBool(vars) in: %s, got: %v, want %v", c.in, got, c.want)
		}
		if err == nil {
			t.Errorf("EvalBool(vars) in: %s, err: nil, wantError %v",
				c.in, c.wantError)
		} else if err != c.wantError {
			t.Errorf("EvalBool(vars) int: %s, err: %v, wantError %v",
				c.in, err, c.wantError)
		}
	}
}

/*************************
 *       Benchmarks
 *************************/
func BenchmarkEvalBool(b *testing.B) {
	b.StopTimer()
	vars := map[string]*dlit.Literal{
		"flowIn":  dlit.MustNew(1.723),
		"flowOut": dlit.MustNew(1.12),
		"name":    dlit.NewString("Fred Wright"),
	}
	funcs := map[string]CallFun{
		"roundto": roundTo,
	}
	benchmarks := []struct {
		expr string
		want bool
	}{
		{expr: "roundto(8+2.25, 1) == 10.3", want: true},
		{expr: "flowIn < flowOut", want: false},
		{expr: "flowIn != 7", want: true},
		{expr: "(flowIn < flowOut) && (flowIn != 7)", want: false},
		{expr: "(flowIn < flowOut) || (flowIn != 7)", want: true},
		{expr: "flowIn > -3", want: true},
		{expr: "name == \"Fred Wright\"", want: true},
		{expr: "[]lit{\"fred\", \"bob\", \"alf\"}[2] == \"alf\"", want: true},
		{expr: "\"Hello world\"[6] == \"h\"", want: false},
		{expr: "9 + (8 + 2) > 18", want: true},
	}
	for _, bm := range benchmarks {
		b.Run(bm.expr, func(b *testing.B) {
			b.StopTimer()
			dexpr, err := New(bm.expr, funcs)
			if err != nil {
				b.Errorf("New: %s", err)
			}
			for n := 0; n < b.N; n++ {
				b.StartTimer()
				got, err := dexpr.EvalBool(vars)
				b.StopTimer()
				if err != nil {
					b.Errorf("EvalBool: ", err)
				}
				if got != bm.want {
					b.Errorf("EvalBool - got: %v, want %v", got, bm.want)
				}
			}
		})
	}
}

/**********************************
 *    Helper functions
 **********************************/
var errTooManyArguments = errors.New("too many arguments")

func roundTo(args []*dlit.Literal) (*dlit.Literal, error) {
	if len(args) > 2 {
		err := errTooManyArguments
		return dlit.MustNew(err), err
	}
	x, isFloat := args[0].Float()
	if !isFloat {
		err := errors.New("can't convert to float")
		return dlit.MustNew(err), err
	}
	p, isInt := args[1].Int()
	if !isInt {
		err := errors.New("can't convert to int")
		return dlit.MustNew(err), err
	}
	// This uses round half-up to tie-break
	shift := math.Pow(10, float64(p))
	return dlit.MustNew(math.Floor(.5+x*shift) / shift), nil
}
