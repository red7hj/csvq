package query

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/mithrandie/csvq/lib/cmd"
	"github.com/mithrandie/csvq/lib/file"
	"github.com/mithrandie/csvq/lib/parser"
	"github.com/mithrandie/csvq/lib/value"
)

var fetchCursorTests = []struct {
	Name          string
	CurName       parser.Identifier
	FetchPosition parser.FetchPosition
	Variables     []parser.Variable
	Success       bool
	ResultVars    VariableMap
	Error         string
}{
	{
		Name:    "Fetch Cursor First Time",
		CurName: parser.Identifier{Literal: "cur"},
		Variables: []parser.Variable{
			{Name: "@var1"},
			{Name: "@var2"},
		},
		Success: true,
		ResultVars: VariableMap{
			"@var1": value.NewString("1"),
			"@var2": value.NewString("str1"),
		},
	},
	{
		Name:    "Fetch Cursor Second Time",
		CurName: parser.Identifier{Literal: "cur"},
		Variables: []parser.Variable{
			{Name: "@var1"},
			{Name: "@var2"},
		},
		Success: true,
		ResultVars: VariableMap{
			"@var1": value.NewString("2"),
			"@var2": value.NewString("str2"),
		},
	},
	{
		Name:    "Fetch Cursor Third Time",
		CurName: parser.Identifier{Literal: "cur"},
		Variables: []parser.Variable{
			{Name: "@var1"},
			{Name: "@var2"},
		},
		Success: true,
		ResultVars: VariableMap{
			"@var1": value.NewString("3"),
			"@var2": value.NewString("str3"),
		},
	},
	{
		Name:    "Fetch Cursor Forth Time",
		CurName: parser.Identifier{Literal: "cur"},
		Variables: []parser.Variable{
			{Name: "@var1"},
			{Name: "@var2"},
		},
		Success: false,
		ResultVars: VariableMap{
			"@var1": value.NewString("3"),
			"@var2": value.NewString("str3"),
		},
	},
	{
		Name:    "Fetch Cursor Absolute",
		CurName: parser.Identifier{Literal: "cur"},
		FetchPosition: parser.FetchPosition{
			Position: parser.Token{Token: parser.ABSOLUTE, Literal: "absolute"},
			Number:   parser.NewIntegerValueFromString("1"),
		},
		Variables: []parser.Variable{
			{Name: "@var1"},
			{Name: "@var2"},
		},
		Success: true,
		ResultVars: VariableMap{
			"@var1": value.NewString("2"),
			"@var2": value.NewString("str2"),
		},
	},
	{
		Name:    "Fetch Cursor Fetch Error",
		CurName: parser.Identifier{Literal: "notexist"},
		Variables: []parser.Variable{
			{Name: "@var1"},
			{Name: "@var2"},
		},
		Error: "[L:- C:-] cursor notexist is undeclared",
	},
	{
		Name:    "Fetch Cursor Not Match Number Error",
		CurName: parser.Identifier{Literal: "cur2"},
		Variables: []parser.Variable{
			{Name: "@var1"},
		},
		Error: "[L:- C:-] fetching from cursor cur2 returns 2 values",
	},
	{
		Name:    "Fetch Cursor Substitution Error",
		CurName: parser.Identifier{Literal: "cur2"},
		Variables: []parser.Variable{
			{Name: "@var1"},
			{Name: "@notexist"},
		},
		Error: "[L:- C:-] variable @notexist is undeclared",
	},
	{
		Name:    "Fetch Cursor Number Value Error",
		CurName: parser.Identifier{Literal: "cur"},
		FetchPosition: parser.FetchPosition{
			Position: parser.Token{Token: parser.ABSOLUTE, Literal: "absolute"},
			Number:   parser.FieldReference{Column: parser.Identifier{Literal: "notexist"}},
		},
		Variables: []parser.Variable{
			{Name: "@var1"},
			{Name: "@var2"},
		},
		Error: "[L:- C:-] field notexist does not exist",
	},
	{
		Name:    "Fetch Cursor Number Not Integer Error",
		CurName: parser.Identifier{Literal: "cur"},
		FetchPosition: parser.FetchPosition{
			Position: parser.Token{Token: parser.ABSOLUTE, Literal: "absolute"},
			Number:   parser.NewNullValueFromString("null"),
		},
		Variables: []parser.Variable{
			{Name: "@var1"},
			{Name: "@var2"},
		},
		Error: "[L:- C:-] fetching position null is not an integer value",
	},
}

func TestFetchCursor(t *testing.T) {
	tf := cmd.GetFlags()
	tf.Repository = TestDir

	filter := NewFilter(
		[]VariableMap{
			{
				"@var1": value.NewNull(),
				"@var2": value.NewNull(),
			},
		},
		[]ViewMap{{}},
		[]CursorMap{
			{
				"CUR": &Cursor{
					query: selectQueryForCursorTest,
				},
				"CUR2": &Cursor{
					query: selectQueryForCursorTest,
				},
			},
		},
		[]UserDefinedFunctionMap{{}},
	)

	ViewCache.Clean()
	filter.Cursors.Open(parser.Identifier{Literal: "cur"}, filter)
	ViewCache.Clean()
	filter.Cursors.Open(parser.Identifier{Literal: "cur2"}, filter)

	for _, v := range fetchCursorTests {
		success, err := FetchCursor(v.CurName, v.FetchPosition, v.Variables, filter)
		if err != nil {
			if len(v.Error) < 1 {
				t.Errorf("%s: unexpected error %q", v.Name, err)
			} else if err.Error() != v.Error {
				t.Errorf("%s: error %q, want error %q", v.Name, err.Error(), v.Error)
			}
			continue
		}
		if 0 < len(v.Error) {
			t.Errorf("%s: no error, want error %q", v.Name, v.Error)
			continue
		}
		if success != v.Success {
			t.Errorf("%s: success = %t, want %t", v.Name, success, v.Success)
		}
		if !reflect.DeepEqual(filter.Variables[0], v.ResultVars) {
			t.Errorf("%s: global vars = %q, want %q", v.Name, filter.Variables[0], v.ResultVars)
		}
	}
}

var declareViewTests = []struct {
	Name    string
	ViewMap ViewMap
	Expr    parser.ViewDeclaration
	Result  ViewMap
	Error   string
}{
	{
		Name: "Declare View",
		Expr: parser.ViewDeclaration{
			View: parser.Identifier{Literal: "tbl"},
			Fields: []parser.QueryExpression{
				parser.Identifier{Literal: "column1"},
				parser.Identifier{Literal: "column2"},
			},
		},
		Result: ViewMap{
			"TBL": {
				FileInfo: &FileInfo{
					Path:             "tbl",
					IsTemporary:      true,
					InitialHeader:    NewHeader("tbl", []string{"column1", "column2"}),
					InitialRecordSet: RecordSet{},
				},
				Header:    NewHeader("tbl", []string{"column1", "column2"}),
				RecordSet: RecordSet{},
			},
		},
	},
	{
		Name: "Declare View Field Duplicate Error",
		Expr: parser.ViewDeclaration{
			View: parser.Identifier{Literal: "tbl"},
			Fields: []parser.QueryExpression{
				parser.Identifier{Literal: "column1"},
				parser.Identifier{Literal: "column1"},
			},
		},
		Error: "[L:- C:-] field name column1 is a duplicate",
	},
	{
		Name: "Declare View From Query",
		Expr: parser.ViewDeclaration{
			View: parser.Identifier{Literal: "tbl"},
			Fields: []parser.QueryExpression{
				parser.Identifier{Literal: "column1"},
				parser.Identifier{Literal: "column2"},
			},
			Query: parser.SelectQuery{
				SelectEntity: parser.SelectEntity{
					SelectClause: parser.SelectClause{
						Fields: []parser.QueryExpression{
							parser.Field{Object: parser.NewIntegerValueFromString("1")},
							parser.Field{Object: parser.NewIntegerValueFromString("2")},
						},
					},
				},
			},
		},
		Result: ViewMap{
			"TBL": {
				FileInfo: &FileInfo{
					Path:          "tbl",
					IsTemporary:   true,
					InitialHeader: NewHeader("tbl", []string{"column1", "column2"}),
					InitialRecordSet: RecordSet{
						NewRecord([]value.Primary{
							value.NewInteger(1),
							value.NewInteger(2),
						}),
					},
				},
				Header: NewHeader("tbl", []string{"column1", "column2"}),
				RecordSet: RecordSet{
					NewRecord([]value.Primary{
						value.NewInteger(1),
						value.NewInteger(2),
					}),
				},
			},
		},
	},
	{
		Name: "Declare View From Query Query Error",
		Expr: parser.ViewDeclaration{
			View: parser.Identifier{Literal: "tbl"},
			Fields: []parser.QueryExpression{
				parser.Identifier{Literal: "column1"},
				parser.Identifier{Literal: "column2"},
			},
			Query: parser.SelectQuery{
				SelectEntity: parser.SelectEntity{
					SelectClause: parser.SelectClause{
						Fields: []parser.QueryExpression{
							parser.Field{Object: parser.NewIntegerValueFromString("1")},
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "notexist"}}},
						},
					},
				},
			},
		},
		Error: "[L:- C:-] field notexist does not exist",
	},
	{
		Name: "Declare View From Query Field Update Error",
		Expr: parser.ViewDeclaration{
			View: parser.Identifier{Literal: "tbl"},
			Fields: []parser.QueryExpression{
				parser.Identifier{Literal: "column1"},
			},
			Query: parser.SelectQuery{
				SelectEntity: parser.SelectEntity{
					SelectClause: parser.SelectClause{
						Fields: []parser.QueryExpression{
							parser.Field{Object: parser.NewIntegerValueFromString("1")},
							parser.Field{Object: parser.NewIntegerValueFromString("2")},
						},
					},
				},
			},
		},
		Error: "[L:- C:-] select query should return exactly 1 field for view tbl",
	},
	{
		Name: "Declare View  From Query Field Duplicate Error",
		Expr: parser.ViewDeclaration{
			View: parser.Identifier{Literal: "tbl"},
			Fields: []parser.QueryExpression{
				parser.Identifier{Literal: "column1"},
				parser.Identifier{Literal: "column1"},
			},
			Query: parser.SelectQuery{
				SelectEntity: parser.SelectEntity{
					SelectClause: parser.SelectClause{
						Fields: []parser.QueryExpression{
							parser.Field{Object: parser.NewIntegerValueFromString("1")},
							parser.Field{Object: parser.NewIntegerValueFromString("2")},
						},
					},
				},
			},
		},
		Error: "[L:- C:-] field name column1 is a duplicate",
	},
	{
		Name: "Declare View Redeclaration Error",
		ViewMap: ViewMap{
			"TBL": {
				FileInfo: &FileInfo{
					Path:        "tbl",
					IsTemporary: true,
				},
			},
		},
		Expr: parser.ViewDeclaration{
			View: parser.Identifier{Literal: "tbl"},
			Fields: []parser.QueryExpression{
				parser.Identifier{Literal: "column1"},
				parser.Identifier{Literal: "column2"},
			},
		},
		Error: "[L:- C:-] view tbl is redeclared",
	},
}

func TestDeclareView(t *testing.T) {
	filter := NewEmptyFilter()

	for _, v := range declareViewTests {
		if v.ViewMap == nil {
			filter.TempViews = []ViewMap{{}}
		} else {
			filter.TempViews = []ViewMap{v.ViewMap}
		}

		err := DeclareView(v.Expr, filter)
		if err != nil {
			if len(v.Error) < 1 {
				t.Errorf("%s: unexpected error %q", v.Name, err)
			} else if err.Error() != v.Error {
				t.Errorf("%s: error %q, want error %q", v.Name, err.Error(), v.Error)
			}
			continue
		}
		if 0 < len(v.Error) {
			t.Errorf("%s: no error, want error %q", v.Name, v.Error)
			continue
		}
		if !reflect.DeepEqual(filter.TempViews[0], v.Result) {
			t.Errorf("%s: view cache = %v, want %v", v.Name, ViewCache, v.Result)
		}
	}
}

var selectTests = []struct {
	Name   string
	Query  parser.SelectQuery
	Result *View
	Error  string
}{
	{
		Name: "Select",
		Query: parser.SelectQuery{
			SelectEntity: parser.SelectEntity{
				SelectClause: parser.SelectClause{
					Fields: []parser.QueryExpression{
						parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column1"}}},
						parser.Field{Object: parser.AggregateFunction{Name: "count", Args: []parser.QueryExpression{parser.AllColumns{}}}},
					},
				},
				FromClause: parser.FromClause{
					Tables: []parser.QueryExpression{
						parser.Table{Object: parser.Identifier{Literal: "group_table"}},
					},
				},
				WhereClause: parser.WhereClause{
					Filter: parser.Comparison{
						LHS:      parser.FieldReference{Column: parser.Identifier{Literal: "column1"}},
						RHS:      parser.NewIntegerValueFromString("3"),
						Operator: "<",
					},
				},
				GroupByClause: parser.GroupByClause{
					Items: []parser.QueryExpression{
						parser.FieldReference{Column: parser.Identifier{Literal: "column1"}},
					},
				},
				HavingClause: parser.HavingClause{
					Filter: parser.Comparison{
						LHS:      parser.AggregateFunction{Name: "count", Args: []parser.QueryExpression{parser.AllColumns{}}},
						RHS:      parser.NewIntegerValueFromString("1"),
						Operator: ">",
					},
				},
			},
			OrderByClause: parser.OrderByClause{
				Items: []parser.QueryExpression{
					parser.OrderItem{Value: parser.FieldReference{Column: parser.Identifier{Literal: "column1"}}},
				},
			},
			LimitClause: parser.LimitClause{
				Value: parser.NewIntegerValueFromString("5"),
			},
			OffsetClause: parser.OffsetClause{
				Value: parser.NewIntegerValue(0),
			},
		},
		Result: &View{
			FileInfo: &FileInfo{
				Path:      GetTestFilePath("group_table.csv"),
				Delimiter: ',',
				NoHeader:  false,
				Encoding:  cmd.UTF8,
				LineBreak: cmd.LF,
			},
			Header: []HeaderField{
				{
					View:        "group_table",
					Column:      "column1",
					Number:      1,
					IsFromTable: true,
				},
				{
					Column:      "count(*)",
					Number:      2,
					IsFromTable: true,
				},
			},
			RecordSet: []Record{
				NewRecord([]value.Primary{
					value.NewString("1"),
					value.NewInteger(2),
				}),
				NewRecord([]value.Primary{
					value.NewString("2"),
					value.NewInteger(2),
				}),
			},
		},
	},
	{
		Name: "Select Replace Columns",
		Query: parser.SelectQuery{
			SelectEntity: parser.SelectEntity{
				SelectClause: parser.SelectClause{
					Fields: []parser.QueryExpression{
						parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column2"}}},
						parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column1"}}},
					},
				},
				FromClause: parser.FromClause{
					Tables: []parser.QueryExpression{
						parser.Table{Object: parser.Identifier{Literal: "table1"}},
					},
				},
			},
			LimitClause: parser.LimitClause{
				Value: parser.NewIntegerValueFromString("1"),
			},
		},
		Result: &View{
			FileInfo: &FileInfo{
				Path:      GetTestFilePath("table1.csv"),
				Delimiter: ',',
				NoHeader:  false,
				Encoding:  cmd.UTF8,
				LineBreak: cmd.LF,
			},
			Header: NewHeader("table1", []string{"column2", "column1"}),
			RecordSet: []Record{
				NewRecord([]value.Primary{
					value.NewString("str1"),
					value.NewString("1"),
				}),
			},
		},
	},
	{
		Name: "Union",
		Query: parser.SelectQuery{
			SelectEntity: parser.SelectSet{
				LHS: parser.SelectEntity{
					SelectClause: parser.SelectClause{
						Fields: []parser.QueryExpression{
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column1"}}},
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column2"}}},
						},
					},
					FromClause: parser.FromClause{
						Tables: []parser.QueryExpression{
							parser.Table{Object: parser.Identifier{Literal: "table1"}},
						},
					},
				},
				Operator: parser.Token{Token: parser.UNION, Literal: "union"},
				RHS: parser.SelectEntity{
					SelectClause: parser.SelectClause{
						Fields: []parser.QueryExpression{
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column3"}}},
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column4"}}},
						},
					},
					FromClause: parser.FromClause{
						Tables: []parser.QueryExpression{
							parser.Table{Object: parser.Identifier{Literal: "table4"}},
						},
					},
				},
			},
		},
		Result: &View{
			Header: NewHeader("table1", []string{"column1", "column2"}),
			RecordSet: []Record{
				NewRecord([]value.Primary{
					value.NewString("1"),
					value.NewString("str1"),
				}),
				NewRecord([]value.Primary{
					value.NewString("2"),
					value.NewString("str2"),
				}),
				NewRecord([]value.Primary{
					value.NewString("3"),
					value.NewString("str3"),
				}),
				NewRecord([]value.Primary{
					value.NewString("4"),
					value.NewString("str4"),
				}),
			},
		},
	},
	{
		Name: "Intersect",
		Query: parser.SelectQuery{
			SelectEntity: parser.SelectSet{
				LHS: parser.SelectEntity{
					SelectClause: parser.SelectClause{
						Fields: []parser.QueryExpression{
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column1"}}},
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column2"}}},
						},
					},
					FromClause: parser.FromClause{
						Tables: []parser.QueryExpression{
							parser.Table{Object: parser.Identifier{Literal: "table1"}},
						},
					},
				},
				Operator: parser.Token{Token: parser.INTERSECT, Literal: "intersect"},
				RHS: parser.SelectEntity{
					SelectClause: parser.SelectClause{
						Fields: []parser.QueryExpression{
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column3"}}},
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column4"}}},
						},
					},
					FromClause: parser.FromClause{
						Tables: []parser.QueryExpression{
							parser.Table{Object: parser.Identifier{Literal: "table4"}},
						},
					},
				},
			},
		},
		Result: &View{
			Header: NewHeader("table1", []string{"column1", "column2"}),
			RecordSet: []Record{
				NewRecord([]value.Primary{
					value.NewString("2"),
					value.NewString("str2"),
				}),
				NewRecord([]value.Primary{
					value.NewString("3"),
					value.NewString("str3"),
				}),
			},
		},
	},
	{
		Name: "Except",
		Query: parser.SelectQuery{
			SelectEntity: parser.SelectSet{
				LHS: parser.SelectEntity{
					SelectClause: parser.SelectClause{
						Fields: []parser.QueryExpression{
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column1"}}},
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column2"}}},
						},
					},
					FromClause: parser.FromClause{
						Tables: []parser.QueryExpression{
							parser.Table{Object: parser.Identifier{Literal: "table1"}},
						},
					},
				},
				Operator: parser.Token{Token: parser.EXCEPT, Literal: "except"},
				RHS: parser.SelectEntity{
					SelectClause: parser.SelectClause{
						Fields: []parser.QueryExpression{
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column3"}}},
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column4"}}},
						},
					},
					FromClause: parser.FromClause{
						Tables: []parser.QueryExpression{
							parser.Table{Object: parser.Identifier{Literal: "table4"}},
						},
					},
				},
			},
		},
		Result: &View{
			Header: NewHeader("table1", []string{"column1", "column2"}),
			RecordSet: []Record{
				NewRecord([]value.Primary{
					value.NewString("1"),
					value.NewString("str1"),
				}),
			},
		},
	},
	{
		Name: "Union with SubQuery",
		Query: parser.SelectQuery{
			SelectEntity: parser.SelectSet{
				LHS: parser.Subquery{
					Query: parser.SelectQuery{
						SelectEntity: parser.SelectEntity{
							SelectClause: parser.SelectClause{
								Fields: []parser.QueryExpression{
									parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column1"}}},
									parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column2"}}},
								},
							},
							FromClause: parser.FromClause{
								Tables: []parser.QueryExpression{
									parser.Table{Object: parser.Identifier{Literal: "table1"}},
								},
							},
						},
					},
				},
				Operator: parser.Token{Token: parser.UNION, Literal: "union"},
				RHS: parser.SelectEntity{
					SelectClause: parser.SelectClause{
						Fields: []parser.QueryExpression{
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column3"}}},
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column4"}}},
						},
					},
					FromClause: parser.FromClause{
						Tables: []parser.QueryExpression{
							parser.Table{Object: parser.Identifier{Literal: "table4"}},
						},
					},
				},
			},
		},
		Result: &View{
			Header: NewHeader("table1", []string{"column1", "column2"}),
			RecordSet: []Record{
				NewRecord([]value.Primary{
					value.NewString("1"),
					value.NewString("str1"),
				}),
				NewRecord([]value.Primary{
					value.NewString("2"),
					value.NewString("str2"),
				}),
				NewRecord([]value.Primary{
					value.NewString("3"),
					value.NewString("str3"),
				}),
				NewRecord([]value.Primary{
					value.NewString("4"),
					value.NewString("str4"),
				}),
			},
		},
	},
	{
		Name: "Union Field Length Error",
		Query: parser.SelectQuery{
			SelectEntity: parser.SelectSet{
				LHS: parser.SelectEntity{
					SelectClause: parser.SelectClause{
						Fields: []parser.QueryExpression{
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column1"}}},
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column2"}}},
						},
					},
					FromClause: parser.FromClause{
						Tables: []parser.QueryExpression{
							parser.Table{Object: parser.Identifier{Literal: "table1"}},
						},
					},
				},
				Operator: parser.Token{Token: parser.UNION, Literal: "union"},
				RHS: parser.SelectEntity{
					SelectClause: parser.SelectClause{
						Fields: []parser.QueryExpression{
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column3"}}},
						},
					},
					FromClause: parser.FromClause{
						Tables: []parser.QueryExpression{
							parser.Table{Object: parser.Identifier{Literal: "table4"}},
						},
					},
				},
			},
		},
		Error: "[L:- C:-] result set to be combined should contain exactly 2 fields",
	},
	{
		Name: "Union LHS Error",
		Query: parser.SelectQuery{
			SelectEntity: parser.SelectSet{
				LHS: parser.SelectEntity{
					SelectClause: parser.SelectClause{
						Fields: []parser.QueryExpression{
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column1"}}},
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "notexist"}}},
						},
					},
					FromClause: parser.FromClause{
						Tables: []parser.QueryExpression{
							parser.Table{Object: parser.Identifier{Literal: "table1"}},
						},
					},
				},
				Operator: parser.Token{Token: parser.UNION, Literal: "union"},
				RHS: parser.SelectEntity{
					SelectClause: parser.SelectClause{
						Fields: []parser.QueryExpression{
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column3"}}},
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column4"}}},
						},
					},
					FromClause: parser.FromClause{
						Tables: []parser.QueryExpression{
							parser.Table{Object: parser.Identifier{Literal: "table4"}},
						},
					},
				},
			},
		},
		Error: "[L:- C:-] field notexist does not exist",
	},
	{
		Name: "Union RHS Error",
		Query: parser.SelectQuery{
			SelectEntity: parser.SelectSet{
				LHS: parser.SelectEntity{
					SelectClause: parser.SelectClause{
						Fields: []parser.QueryExpression{
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column1"}}},
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column2"}}},
						},
					},
					FromClause: parser.FromClause{
						Tables: []parser.QueryExpression{
							parser.Table{Object: parser.Identifier{Literal: "table1"}},
						},
					},
				},
				Operator: parser.Token{Token: parser.UNION, Literal: "union"},
				RHS: parser.SelectEntity{
					SelectClause: parser.SelectClause{
						Fields: []parser.QueryExpression{
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column3"}}},
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "notexist"}}},
						},
					},
					FromClause: parser.FromClause{
						Tables: []parser.QueryExpression{
							parser.Table{Object: parser.Identifier{Literal: "table4"}},
						},
					},
				},
			},
		},
		Error: "[L:- C:-] field notexist does not exist",
	},
	{
		Name: "Inline Tables",
		Query: parser.SelectQuery{
			WithClause: parser.WithClause{
				With: "with",
				InlineTables: []parser.QueryExpression{
					parser.InlineTable{
						Name: parser.Identifier{Literal: "it"},
						Fields: []parser.QueryExpression{
							parser.Identifier{Literal: "c1"},
						},
						As: "as",
						Query: parser.SelectQuery{
							SelectEntity: parser.SelectEntity{
								SelectClause: parser.SelectClause{
									Select: "select",
									Fields: []parser.QueryExpression{
										parser.Field{Object: parser.NewIntegerValueFromString("2")},
									},
								},
							},
						},
					},
				},
			},
			SelectEntity: parser.SelectEntity{
				SelectClause: parser.SelectClause{
					Fields: []parser.QueryExpression{
						parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "c1"}}},
					},
				},
				FromClause: parser.FromClause{
					Tables: []parser.QueryExpression{
						parser.Table{Object: parser.Identifier{Literal: "it"}},
					},
				},
			},
		},
		Result: &View{
			Header: NewHeader("it", []string{"c1"}),
			RecordSet: []Record{
				NewRecord([]value.Primary{
					value.NewInteger(2),
				}),
			},
		},
	},
	{
		Name: "Inline Tables Field Length Error",
		Query: parser.SelectQuery{
			WithClause: parser.WithClause{
				With: "with",
				InlineTables: []parser.QueryExpression{
					parser.InlineTable{
						Name: parser.Identifier{Literal: "it"},
						Fields: []parser.QueryExpression{
							parser.Identifier{Literal: "c1"},
						},
						As: "as",
						Query: parser.SelectQuery{
							SelectEntity: parser.SelectSet{
								LHS: parser.SelectEntity{
									SelectClause: parser.SelectClause{
										Fields: []parser.QueryExpression{
											parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column1"}}},
											parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column2"}}},
										},
									},
									FromClause: parser.FromClause{
										Tables: []parser.QueryExpression{
											parser.Table{Object: parser.Identifier{Literal: "table1"}},
										},
									},
								},
								Operator: parser.Token{Token: parser.UNION, Literal: "union"},
								RHS: parser.SelectEntity{
									SelectClause: parser.SelectClause{
										Fields: []parser.QueryExpression{
											parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column3"}}},
											parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column4"}}},
										},
									},
									FromClause: parser.FromClause{
										Tables: []parser.QueryExpression{
											parser.Table{Object: parser.Identifier{Literal: "table4"}},
										},
									},
								},
							},
						},
					},
				},
			},
			SelectEntity: parser.SelectEntity{
				SelectClause: parser.SelectClause{
					Fields: []parser.QueryExpression{
						parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "c1"}}},
					},
				},
				FromClause: parser.FromClause{
					Tables: []parser.QueryExpression{
						parser.Table{Object: parser.Identifier{Literal: "it"}},
					},
				},
			},
		},
		Error: "[L:- C:-] select query should return exactly 1 field for inline table it",
	},
	{
		Name: "Inline Tables Recursion",
		Query: parser.SelectQuery{
			WithClause: parser.WithClause{
				With: "with",
				InlineTables: []parser.QueryExpression{
					parser.InlineTable{
						Recursive: parser.Token{Token: parser.RECURSIVE, Literal: "recursive"},
						Name:      parser.Identifier{Literal: "it"},
						Fields: []parser.QueryExpression{
							parser.Identifier{Literal: "n"},
						},
						As: "as",
						Query: parser.SelectQuery{
							SelectEntity: parser.SelectSet{
								LHS: parser.SelectEntity{
									SelectClause: parser.SelectClause{
										Select: "select",
										Fields: []parser.QueryExpression{
											parser.Field{Object: parser.NewIntegerValueFromString("1")},
										},
									},
								},
								Operator: parser.Token{Token: parser.UNION, Literal: "union"},
								RHS: parser.SelectEntity{
									SelectClause: parser.SelectClause{
										Select: "select",
										Fields: []parser.QueryExpression{
											parser.Field{
												Object: parser.Arithmetic{
													LHS:      parser.FieldReference{Column: parser.Identifier{Literal: "n"}},
													RHS:      parser.NewIntegerValueFromString("1"),
													Operator: '+',
												},
											},
										},
									},
									FromClause: parser.FromClause{
										Tables: []parser.QueryExpression{
											parser.Table{Object: parser.Identifier{Literal: "it"}},
										},
									},
									WhereClause: parser.WhereClause{
										Filter: parser.Comparison{
											LHS:      parser.FieldReference{Column: parser.Identifier{Literal: "n"}},
											RHS:      parser.NewIntegerValueFromString("3"),
											Operator: "<",
										},
									},
								},
							},
						},
					},
				},
			},
			SelectEntity: parser.SelectEntity{
				SelectClause: parser.SelectClause{
					Fields: []parser.QueryExpression{
						parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "n"}}},
					},
				},
				FromClause: parser.FromClause{
					Tables: []parser.QueryExpression{
						parser.Table{Object: parser.Identifier{Literal: "it"}},
					},
				},
			},
		},
		Result: &View{
			Header: []HeaderField{
				{
					View:        "it",
					Column:      "n",
					Number:      1,
					IsFromTable: true,
				},
			},
			RecordSet: []Record{
				NewRecord([]value.Primary{
					value.NewInteger(1),
				}),
				NewRecord([]value.Primary{
					value.NewInteger(2),
				}),
				NewRecord([]value.Primary{
					value.NewInteger(3),
				}),
			},
		},
	},
	{
		Name: "Inline Tables Recursion Field Length Error",
		Query: parser.SelectQuery{
			WithClause: parser.WithClause{
				With: "with",
				InlineTables: []parser.QueryExpression{
					parser.InlineTable{
						Recursive: parser.Token{Token: parser.RECURSIVE, Literal: "recursive"},
						Name:      parser.Identifier{Literal: "it"},
						Fields: []parser.QueryExpression{
							parser.Identifier{Literal: "n"},
						},
						As: "as",
						Query: parser.SelectQuery{
							SelectEntity: parser.SelectSet{
								LHS: parser.SelectEntity{
									SelectClause: parser.SelectClause{
										Select: "select",
										Fields: []parser.QueryExpression{
											parser.Field{Object: parser.NewIntegerValueFromString("1")},
										},
									},
								},
								Operator: parser.Token{Token: parser.UNION, Literal: "union"},
								RHS: parser.SelectEntity{
									SelectClause: parser.SelectClause{
										Select: "select",
										Fields: []parser.QueryExpression{
											parser.Field{
												Object: parser.Arithmetic{
													LHS:      parser.FieldReference{Column: parser.Identifier{Literal: "n"}},
													RHS:      parser.NewIntegerValueFromString("1"),
													Operator: '+',
												},
											},
											parser.Field{Object: parser.NewIntegerValueFromString("2")},
										},
									},
									FromClause: parser.FromClause{
										Tables: []parser.QueryExpression{
											parser.Table{Object: parser.Identifier{Literal: "it"}},
										},
									},
									WhereClause: parser.WhereClause{
										Filter: parser.Comparison{
											LHS:      parser.FieldReference{Column: parser.Identifier{Literal: "n"}},
											RHS:      parser.NewIntegerValueFromString("3"),
											Operator: "<",
										},
									},
								},
							},
						},
					},
				},
			},
			SelectEntity: parser.SelectEntity{
				SelectClause: parser.SelectClause{
					Fields: []parser.QueryExpression{
						parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "n"}}},
					},
				},
				FromClause: parser.FromClause{
					Tables: []parser.QueryExpression{
						parser.Table{Object: parser.Identifier{Literal: "it"}},
					},
				},
			},
		},
		Error: "[L:- C:-] result set to be combined should contain exactly 1 field",
	},
}

func TestSelect(t *testing.T) {
	tf := cmd.GetFlags()
	tf.Repository = TestDir

	filter := NewEmptyFilter()

	for _, v := range selectTests {
		ViewCache.Clean()
		result, err := Select(v.Query, filter)
		if err != nil {
			if len(v.Error) < 1 {
				t.Errorf("%s: unexpected error %q", v.Name, err)
			} else if err.Error() != v.Error {
				t.Errorf("%s: error %q, want error %q", v.Name, err.Error(), v.Error)
			}
			continue
		}
		if 0 < len(v.Error) {
			t.Errorf("%s: no error, want error %q", v.Name, v.Error)
			continue
		}
		if !reflect.DeepEqual(result, v.Result) {
			t.Errorf("%s: result = %v, want %v", v.Name, result, v.Result)
		}
	}
}

var insertTests = []struct {
	Name         string
	Query        parser.InsertQuery
	Result       *View
	ViewCache    ViewMap
	TempViewList TemporaryViewScopes
	Error        string
}{
	{
		Name: "Insert Query",
		Query: parser.InsertQuery{
			WithClause: parser.WithClause{
				With: "with",
				InlineTables: []parser.QueryExpression{
					parser.InlineTable{
						Name: parser.Identifier{Literal: "it"},
						Fields: []parser.QueryExpression{
							parser.Identifier{Literal: "c1"},
						},
						As: "as",
						Query: parser.SelectQuery{
							SelectEntity: parser.SelectEntity{
								SelectClause: parser.SelectClause{
									Select: "select",
									Fields: []parser.QueryExpression{
										parser.Field{Object: parser.NewIntegerValueFromString("2")},
									},
								},
							},
						},
					},
				},
			},
			Table: parser.Table{Object: parser.Identifier{Literal: "table1"}},
			Fields: []parser.QueryExpression{
				parser.FieldReference{Column: parser.Identifier{Literal: "column1"}},
			},
			ValuesList: []parser.QueryExpression{
				parser.RowValue{
					Value: parser.ValueList{
						Values: []parser.QueryExpression{
							parser.NewIntegerValueFromString("4"),
						},
					},
				},
				parser.RowValue{
					Value: parser.ValueList{
						Values: []parser.QueryExpression{
							parser.Subquery{
								Query: parser.SelectQuery{
									SelectEntity: parser.SelectEntity{
										SelectClause: parser.SelectClause{
											Select: "select",
											Fields: []parser.QueryExpression{
												parser.Field{Object: parser.FieldReference{View: parser.Identifier{Literal: "it"}, Column: parser.Identifier{Literal: "c1"}}},
											},
										},
										FromClause: parser.FromClause{
											Tables: []parser.QueryExpression{
												parser.Table{Object: parser.Identifier{Literal: "it"}},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		Result: &View{
			FileInfo: &FileInfo{
				Path:      GetTestFilePath("table1.csv"),
				Delimiter: ',',
				NoHeader:  false,
				Encoding:  cmd.UTF8,
				LineBreak: cmd.LF,
			},
			Header: NewHeader("table1", []string{"column1", "column2"}),
			RecordSet: []Record{
				NewRecord([]value.Primary{
					value.NewString("1"),
					value.NewString("str1"),
				}),
				NewRecord([]value.Primary{
					value.NewString("2"),
					value.NewString("str2"),
				}),
				NewRecord([]value.Primary{
					value.NewString("3"),
					value.NewString("str3"),
				}),
				NewRecord([]value.Primary{
					value.NewInteger(4),
					value.NewNull(),
				}),
				NewRecord([]value.Primary{
					value.NewInteger(2),
					value.NewNull(),
				}),
			},
			ForUpdate:       true,
			OperatedRecords: 2,
		},
		ViewCache: ViewMap{
			strings.ToUpper(GetTestFilePath("table1.csv")): &View{
				FileInfo: &FileInfo{
					Path:      GetTestFilePath("table1.csv"),
					Delimiter: ',',
					NoHeader:  false,
					Encoding:  cmd.UTF8,
					LineBreak: cmd.LF,
				},
				Header: NewHeader("table1", []string{"column1", "column2"}),
				RecordSet: []Record{
					NewRecord([]value.Primary{
						value.NewString("1"),
						value.NewString("str1"),
					}),
					NewRecord([]value.Primary{
						value.NewString("2"),
						value.NewString("str2"),
					}),
					NewRecord([]value.Primary{
						value.NewString("3"),
						value.NewString("str3"),
					}),
					NewRecord([]value.Primary{
						value.NewInteger(4),
						value.NewNull(),
					}),
					NewRecord([]value.Primary{
						value.NewInteger(2),
						value.NewNull(),
					}),
				},
				ForUpdate:       true,
				OperatedRecords: 2,
			},
		},
	},
	{
		Name: "Insert Query For Temporary View",
		Query: parser.InsertQuery{
			Table: parser.Table{Object: parser.Identifier{Literal: "tmpview"}, Alias: parser.Identifier{Literal: "t"}},
			Fields: []parser.QueryExpression{
				parser.FieldReference{Column: parser.Identifier{Literal: "column1"}},
			},
			ValuesList: []parser.QueryExpression{
				parser.RowValue{
					Value: parser.ValueList{
						Values: []parser.QueryExpression{
							parser.NewIntegerValueFromString("4"),
						},
					},
				},
				parser.RowValue{
					Value: parser.ValueList{
						Values: []parser.QueryExpression{
							parser.NewIntegerValueFromString("2"),
						},
					},
				},
			},
		},
		Result: &View{
			FileInfo: &FileInfo{
				Path:        "tmpview",
				Delimiter:   ',',
				IsTemporary: true,
			},
			Header: NewHeader("tmpview", []string{"column1", "column2"}),
			RecordSet: []Record{
				NewRecord([]value.Primary{
					value.NewString("1"),
					value.NewString("str1"),
				}),
				NewRecord([]value.Primary{
					value.NewString("2"),
					value.NewString("str2"),
				}),
				NewRecord([]value.Primary{
					value.NewInteger(4),
					value.NewNull(),
				}),
				NewRecord([]value.Primary{
					value.NewInteger(2),
					value.NewNull(),
				}),
			},
			ForUpdate:       true,
			OperatedRecords: 2,
		},
		TempViewList: TemporaryViewScopes{
			ViewMap{
				"TMPVIEW": &View{
					Header: NewHeader("tmpview", []string{"column1", "column2"}),
					RecordSet: []Record{
						NewRecord([]value.Primary{
							value.NewString("1"),
							value.NewString("str1"),
						}),
						NewRecord([]value.Primary{
							value.NewString("2"),
							value.NewString("str2"),
						}),
						NewRecord([]value.Primary{
							value.NewInteger(4),
							value.NewNull(),
						}),
						NewRecord([]value.Primary{
							value.NewInteger(2),
							value.NewNull(),
						}),
					},
					FileInfo: &FileInfo{
						Path:        "tmpview",
						Delimiter:   ',',
						IsTemporary: true,
					},
					ForUpdate:       true,
					OperatedRecords: 2,
				},
			},
		},
	},
	{
		Name: "Insert Query All Columns",
		Query: parser.InsertQuery{
			Table: parser.Table{Object: parser.Identifier{Literal: "table1"}},
			ValuesList: []parser.QueryExpression{
				parser.RowValue{
					Value: parser.ValueList{
						Values: []parser.QueryExpression{
							parser.NewIntegerValueFromString("4"),
							parser.NewStringValue("str4"),
						},
					},
				},
				parser.RowValue{
					Value: parser.ValueList{
						Values: []parser.QueryExpression{
							parser.NewIntegerValueFromString("5"),
							parser.NewStringValue("str5"),
						},
					},
				},
			},
		},
		Result: &View{
			FileInfo: &FileInfo{
				Path:      GetTestFilePath("table1.csv"),
				Delimiter: ',',
				NoHeader:  false,
				Encoding:  cmd.UTF8,
				LineBreak: cmd.LF,
			},
			Header: NewHeader("table1", []string{"column1", "column2"}),
			RecordSet: []Record{
				NewRecord([]value.Primary{
					value.NewString("1"),
					value.NewString("str1"),
				}),
				NewRecord([]value.Primary{
					value.NewString("2"),
					value.NewString("str2"),
				}),
				NewRecord([]value.Primary{
					value.NewString("3"),
					value.NewString("str3"),
				}),
				NewRecord([]value.Primary{
					value.NewInteger(4),
					value.NewString("str4"),
				}),
				NewRecord([]value.Primary{
					value.NewInteger(5),
					value.NewString("str5"),
				}),
			},
			ForUpdate:       true,
			OperatedRecords: 2,
		},
	},
	{
		Name: "Insert Query File Does Not Exist Error",
		Query: parser.InsertQuery{
			Table: parser.Table{Object: parser.Identifier{Literal: "notexist"}},
			Fields: []parser.QueryExpression{
				parser.FieldReference{Column: parser.Identifier{Literal: "column1"}},
			},
			ValuesList: []parser.QueryExpression{
				parser.RowValue{
					Value: parser.ValueList{
						Values: []parser.QueryExpression{
							parser.NewIntegerValueFromString("4"),
						},
					},
				},
				parser.RowValue{
					Value: parser.ValueList{
						Values: []parser.QueryExpression{
							parser.NewIntegerValueFromString("5"),
						},
					},
				},
			},
		},
		Error: "[L:- C:-] file notexist does not exist",
	},
	{
		Name: "Insert Query Field Does Not Exist Error",
		Query: parser.InsertQuery{
			Table: parser.Table{Object: parser.Identifier{Literal: "table1"}},
			Fields: []parser.QueryExpression{
				parser.FieldReference{Column: parser.Identifier{Literal: "notexist"}},
			},
			ValuesList: []parser.QueryExpression{
				parser.RowValue{
					Value: parser.ValueList{
						Values: []parser.QueryExpression{
							parser.NewIntegerValueFromString("4"),
						},
					},
				},
				parser.RowValue{
					Value: parser.ValueList{
						Values: []parser.QueryExpression{
							parser.NewIntegerValueFromString("5"),
						},
					},
				},
			},
		},
		Error: "[L:- C:-] field notexist does not exist",
	},
	{
		Name: "Insert Select Query",
		Query: parser.InsertQuery{
			Table: parser.Table{Object: parser.Identifier{Literal: "table1"}},
			Fields: []parser.QueryExpression{
				parser.FieldReference{Column: parser.Identifier{Literal: "column1"}},
				parser.FieldReference{Column: parser.Identifier{Literal: "column2"}},
			},
			Query: parser.SelectQuery{
				SelectEntity: parser.SelectEntity{
					SelectClause: parser.SelectClause{
						Fields: []parser.QueryExpression{
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column3"}}},
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column4"}}},
						},
					},
					FromClause: parser.FromClause{
						Tables: []parser.QueryExpression{
							parser.Table{Object: parser.Identifier{Literal: "table2"}},
						},
					},
				},
			},
		},
		Result: &View{
			FileInfo: &FileInfo{
				Path:      GetTestFilePath("table1.csv"),
				Delimiter: ',',
				NoHeader:  false,
				Encoding:  cmd.UTF8,
				LineBreak: cmd.LF,
			},
			Header: NewHeader("table1", []string{"column1", "column2"}),
			RecordSet: []Record{
				NewRecord([]value.Primary{
					value.NewString("1"),
					value.NewString("str1"),
				}),
				NewRecord([]value.Primary{
					value.NewString("2"),
					value.NewString("str2"),
				}),
				NewRecord([]value.Primary{
					value.NewString("3"),
					value.NewString("str3"),
				}),
				NewRecord([]value.Primary{
					value.NewString("2"),
					value.NewString("str22"),
				}),
				NewRecord([]value.Primary{
					value.NewString("3"),
					value.NewString("str33"),
				}),
				NewRecord([]value.Primary{
					value.NewString("4"),
					value.NewString("str44"),
				}),
			},
			ForUpdate:       true,
			OperatedRecords: 3,
		},
	},
	{
		Name: "Insert Select Query Field Does Not Exist Error",
		Query: parser.InsertQuery{
			Table: parser.Table{Object: parser.Identifier{Literal: "table1"}},
			Fields: []parser.QueryExpression{
				parser.FieldReference{Column: parser.Identifier{Literal: "column1"}},
			},
			Query: parser.SelectQuery{
				SelectEntity: parser.SelectEntity{
					SelectClause: parser.SelectClause{
						Fields: []parser.QueryExpression{
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column3"}}},
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "column4"}}},
						},
					},
					FromClause: parser.FromClause{
						Tables: []parser.QueryExpression{
							parser.Table{Object: parser.Identifier{Literal: "table2"}},
						},
					},
				},
			},
		},
		Error: "[L:- C:-] select query should return exactly 1 field",
	},
}

func TestInsert(t *testing.T) {
	tf := cmd.GetFlags()
	tf.Repository = TestDir
	tf.Quiet = false

	filter := NewEmptyFilter()
	filter.TempViews = TemporaryViewScopes{
		ViewMap{
			"TMPVIEW": &View{
				Header: NewHeader("tmpview", []string{"column1", "column2"}),
				RecordSet: []Record{
					NewRecord([]value.Primary{
						value.NewString("1"),
						value.NewString("str1"),
					}),
					NewRecord([]value.Primary{
						value.NewString("2"),
						value.NewString("str2"),
					}),
				},
				FileInfo: &FileInfo{
					Path:        "tmpview",
					Delimiter:   ',',
					IsTemporary: true,
				},
			},
		},
	}

	for _, v := range insertTests {
		ReleaseResources()
		result, err := Insert(v.Query, filter)
		if err != nil {
			if len(v.Error) < 1 {
				t.Errorf("%s: unexpected error %q", v.Name, err)
			} else if err.Error() != v.Error {
				t.Errorf("%s: error %q, want error %q", v.Name, err.Error(), v.Error)
			}
			continue
		}
		if 0 < len(v.Error) {
			t.Errorf("%s: no error, want error %q", v.Name, v.Error)
			continue
		}

		for _, v2 := range ViewCache {
			if v2.FileInfo.File != nil {
				if v2.FileInfo.Path != v2.FileInfo.File.Name() {
					t.Errorf("file pointer = %q, want %q for %q", v2.FileInfo.File.Name(), v2.FileInfo.Path, v.Name)
				}
				file.Close(v2.FileInfo.File)
				v2.FileInfo.File = nil
			}
		}

		if !reflect.DeepEqual(result, v.Result) {
			t.Errorf("%s: result = %v, want %v", v.Name, result, v.Result)
		}

		if v.ViewCache != nil {
			if !reflect.DeepEqual(ViewCache, v.ViewCache) {
				t.Errorf("%s: view cache = %v, want %v", v.Name, ViewCache, v.ViewCache)
			}
		}
		if v.TempViewList != nil {
			if !reflect.DeepEqual(filter.TempViews, v.TempViewList) {
				t.Errorf("%s: temporary views list = %v, want %v", v.Name, filter.TempViews, v.TempViewList)
			}
		}
	}
	ReleaseResources()
}

var updateTests = []struct {
	Name         string
	Query        parser.UpdateQuery
	Result       []*View
	ViewCache    ViewMap
	TempViewList TemporaryViewScopes
	Error        string
}{
	{
		Name: "Update Query",
		Query: parser.UpdateQuery{
			WithClause: parser.WithClause{
				With: "with",
				InlineTables: []parser.QueryExpression{
					parser.InlineTable{
						Name: parser.Identifier{Literal: "it"},
						Fields: []parser.QueryExpression{
							parser.Identifier{Literal: "c1"},
						},
						As: "as",
						Query: parser.SelectQuery{
							SelectEntity: parser.SelectEntity{
								SelectClause: parser.SelectClause{
									Select: "select",
									Fields: []parser.QueryExpression{
										parser.Field{Object: parser.NewIntegerValueFromString("2")},
									},
								},
							},
						},
					},
				},
			},
			Tables: []parser.QueryExpression{
				parser.Table{Object: parser.Identifier{Literal: "table1"}},
			},
			SetList: []parser.UpdateSet{
				{
					Field: parser.FieldReference{Column: parser.Identifier{Literal: "column1"}},
					Value: parser.NewStringValue("update1"),
				},
				{
					Field: parser.FieldReference{Column: parser.Identifier{Literal: "column2"}},
					Value: parser.NewStringValue("update2"),
				},
			},
			WhereClause: parser.WhereClause{
				Filter: parser.Comparison{
					LHS: parser.FieldReference{Column: parser.Identifier{Literal: "column1"}},
					RHS: parser.Subquery{
						Query: parser.SelectQuery{
							SelectEntity: parser.SelectEntity{
								SelectClause: parser.SelectClause{
									Select: "select",
									Fields: []parser.QueryExpression{
										parser.Field{Object: parser.FieldReference{View: parser.Identifier{Literal: "it"}, Column: parser.Identifier{Literal: "c1"}}},
									},
								},
								FromClause: parser.FromClause{
									Tables: []parser.QueryExpression{
										parser.Table{Object: parser.Identifier{Literal: "it"}},
									},
								},
							},
						},
					},
					Operator: "=",
				},
			},
		},
		Result: []*View{
			{
				FileInfo: &FileInfo{
					Path:      GetTestFilePath("table1.csv"),
					Delimiter: ',',
					NoHeader:  false,
					Encoding:  cmd.UTF8,
					LineBreak: cmd.LF,
				},
				Header: NewHeader("table1", []string{"column1", "column2"}),
				RecordSet: []Record{
					NewRecord([]value.Primary{
						value.NewString("1"),
						value.NewString("str1"),
					}),
					NewRecord([]value.Primary{
						value.NewString("update1"),
						value.NewString("update2"),
					}),
					NewRecord([]value.Primary{
						value.NewString("3"),
						value.NewString("str3"),
					}),
				},
				ForUpdate:       true,
				OperatedRecords: 1,
			},
		},
		ViewCache: ViewMap{
			strings.ToUpper(GetTestFilePath("table1.csv")): &View{
				FileInfo: &FileInfo{
					Path:      GetTestFilePath("table1.csv"),
					Delimiter: ',',
					NoHeader:  false,
					Encoding:  cmd.UTF8,
					LineBreak: cmd.LF,
				},
				Header: NewHeader("table1", []string{"column1", "column2"}),
				RecordSet: []Record{
					NewRecord([]value.Primary{
						value.NewString("1"),
						value.NewString("str1"),
					}),
					NewRecord([]value.Primary{
						value.NewString("update1"),
						value.NewString("update2"),
					}),
					NewRecord([]value.Primary{
						value.NewString("3"),
						value.NewString("str3"),
					}),
				},
				ForUpdate:       true,
				OperatedRecords: 1,
			},
		},
	},
	{
		Name: "Update Query For Temporary View",
		Query: parser.UpdateQuery{
			Tables: []parser.QueryExpression{
				parser.Table{Object: parser.Identifier{Literal: "tmpview"}, Alias: parser.Identifier{Literal: "t1"}},
			},
			SetList: []parser.UpdateSet{
				{
					Field: parser.ColumnNumber{View: parser.Identifier{Literal: "t1"}, Number: value.NewInteger(2)},
					Value: parser.NewStringValue("update"),
				},
			},
		},
		Result: []*View{
			{
				FileInfo: &FileInfo{
					Path:        "tmpview",
					Delimiter:   ',',
					IsTemporary: true,
				},
				Header: NewHeader("tmpview", []string{"column1", "column2"}),
				RecordSet: []Record{
					NewRecord([]value.Primary{
						value.NewString("1"),
						value.NewString("update"),
					}),
					NewRecord([]value.Primary{
						value.NewString("2"),
						value.NewString("update"),
					}),
				},
				OperatedRecords: 2,
			},
		},
		TempViewList: TemporaryViewScopes{
			ViewMap{
				"TMPVIEW": &View{
					Header: NewHeader("tmpview", []string{"column1", "column2"}),
					RecordSet: []Record{
						NewRecord([]value.Primary{
							value.NewString("1"),
							value.NewString("update"),
						}),
						NewRecord([]value.Primary{
							value.NewString("2"),
							value.NewString("update"),
						}),
					},
					FileInfo: &FileInfo{
						Path:        "tmpview",
						Delimiter:   ',',
						IsTemporary: true,
					},
					OperatedRecords: 2,
				},
			},
		},
	},
	{
		Name: "Update Query Multiple Table",
		Query: parser.UpdateQuery{
			Tables: []parser.QueryExpression{
				parser.Table{Object: parser.Identifier{Literal: "t1"}},
			},
			SetList: []parser.UpdateSet{
				{
					Field: parser.FieldReference{Column: parser.Identifier{Literal: "column2"}},
					Value: parser.FieldReference{Column: parser.Identifier{Literal: "column4"}},
				},
			},
			FromClause: parser.FromClause{
				Tables: []parser.QueryExpression{
					parser.Table{Object: parser.Join{
						Table: parser.Table{
							Object: parser.Identifier{Literal: "table1"},
							Alias:  parser.Identifier{Literal: "t1"},
						},
						JoinTable: parser.Table{
							Object: parser.Identifier{Literal: "table2"},
							Alias:  parser.Identifier{Literal: "t2"},
						},
						Condition: parser.JoinCondition{
							On: parser.Comparison{
								LHS:      parser.FieldReference{Column: parser.Identifier{Literal: "column1"}},
								RHS:      parser.FieldReference{Column: parser.Identifier{Literal: "column3"}},
								Operator: "=",
							},
						},
					}},
				},
			},
		},
		Result: []*View{
			{
				FileInfo: &FileInfo{
					Path:      GetTestFilePath("table1.csv"),
					Delimiter: ',',
					NoHeader:  false,
					Encoding:  cmd.UTF8,
					LineBreak: cmd.LF,
				},
				Header: NewHeader("table1", []string{"column1", "column2"}),
				RecordSet: []Record{
					NewRecord([]value.Primary{
						value.NewString("1"),
						value.NewString("str1"),
					}),
					NewRecord([]value.Primary{
						value.NewString("2"),
						value.NewString("str22"),
					}),
					NewRecord([]value.Primary{
						value.NewString("3"),
						value.NewString("str33"),
					}),
				},
				ForUpdate:       true,
				OperatedRecords: 2,
			},
		},
	},
	{
		Name: "Update Query File Does Not Exist Error",
		Query: parser.UpdateQuery{
			Tables: []parser.QueryExpression{
				parser.Table{Object: parser.Identifier{Literal: "notexist"}},
			},
			SetList: []parser.UpdateSet{
				{
					Field: parser.FieldReference{Column: parser.Identifier{Literal: "column2"}},
					Value: parser.NewStringValue("update"),
				},
			},
			WhereClause: parser.WhereClause{
				Filter: parser.Comparison{
					LHS:      parser.Identifier{Literal: "column1"},
					RHS:      parser.NewIntegerValueFromString("2"),
					Operator: "=",
				},
			},
		},
		Error: "[L:- C:-] file notexist does not exist",
	},
	{
		Name: "Update Query Filter Error",
		Query: parser.UpdateQuery{
			Tables: []parser.QueryExpression{
				parser.Table{Object: parser.Identifier{Literal: "table1"}},
			},
			SetList: []parser.UpdateSet{
				{
					Field: parser.FieldReference{Column: parser.Identifier{Literal: "column1"}},
					Value: parser.NewStringValue("update"),
				},
			},
			WhereClause: parser.WhereClause{
				Filter: parser.Comparison{
					LHS:      parser.FieldReference{Column: parser.Identifier{Literal: "notexist"}},
					RHS:      parser.NewIntegerValueFromString("2"),
					Operator: "=",
				},
			},
		},
		Error: "[L:- C:-] field notexist does not exist",
	},
	{
		Name: "Update Query File Is Not Loaded Error",
		Query: parser.UpdateQuery{
			Tables: []parser.QueryExpression{
				parser.Table{Object: parser.Identifier{Literal: "notexist"}},
			},
			SetList: []parser.UpdateSet{
				{
					Field: parser.FieldReference{Column: parser.Identifier{Literal: "column2"}},
					Value: parser.FieldReference{Column: parser.Identifier{Literal: "column4"}},
				},
			},
			FromClause: parser.FromClause{
				Tables: []parser.QueryExpression{
					parser.Table{Object: parser.Join{
						Table: parser.Table{
							Object: parser.Identifier{Literal: "table1"},
							Alias:  parser.Identifier{Literal: "t1"},
						},
						JoinTable: parser.Table{
							Object: parser.Identifier{Literal: "table2"},
							Alias:  parser.Identifier{Literal: "t2"},
						},
						Condition: parser.JoinCondition{
							On: parser.Comparison{
								LHS:      parser.FieldReference{Column: parser.Identifier{Literal: "column1"}},
								RHS:      parser.FieldReference{Column: parser.Identifier{Literal: "column3"}},
								Operator: "=",
							},
						},
					}},
				},
			},
		},
		Error: "[L:- C:-] table notexist is not loaded",
	},
	{
		Name: "Update Query Update Table Is Not Specified Error",
		Query: parser.UpdateQuery{
			Tables: []parser.QueryExpression{
				parser.Table{Object: parser.Identifier{Literal: "t2"}},
			},
			SetList: []parser.UpdateSet{
				{
					Field: parser.FieldReference{View: parser.Identifier{Literal: "t1"}, Column: parser.Identifier{Literal: "column2"}},
					Value: parser.FieldReference{Column: parser.Identifier{Literal: "column4"}},
				},
			},
			FromClause: parser.FromClause{
				Tables: []parser.QueryExpression{
					parser.Table{Object: parser.Join{
						Table: parser.Table{
							Object: parser.Identifier{Literal: "table1"},
							Alias:  parser.Identifier{Literal: "t1"},
						},
						JoinTable: parser.Table{
							Object: parser.Identifier{Literal: "table2"},
							Alias:  parser.Identifier{Literal: "t2"},
						},
						Condition: parser.JoinCondition{
							On: parser.Comparison{
								LHS:      parser.FieldReference{Column: parser.Identifier{Literal: "column1"}},
								RHS:      parser.FieldReference{Column: parser.Identifier{Literal: "column3"}},
								Operator: "=",
							},
						},
					}},
				},
			},
		},
		Error: "[L:- C:-] field t1.column2 does not exist in the tables to update",
	},
	{
		Name: "Update Query Update Field Error",
		Query: parser.UpdateQuery{
			Tables: []parser.QueryExpression{
				parser.Table{Object: parser.Identifier{Literal: "table1"}},
			},
			SetList: []parser.UpdateSet{
				{
					Field: parser.FieldReference{Column: parser.Identifier{Literal: "notexist"}},
					Value: parser.NewStringValue("update"),
				},
			},
			WhereClause: parser.WhereClause{
				Filter: parser.Comparison{
					LHS:      parser.FieldReference{Column: parser.Identifier{Literal: "column1"}},
					RHS:      parser.NewIntegerValueFromString("2"),
					Operator: "=",
				},
			},
		},
		Error: "[L:- C:-] field notexist does not exist",
	},
	{
		Name: "Update Query Update Value Error",
		Query: parser.UpdateQuery{
			Tables: []parser.QueryExpression{
				parser.Table{Object: parser.Identifier{Literal: "table1"}},
			},
			SetList: []parser.UpdateSet{
				{
					Field: parser.FieldReference{Column: parser.Identifier{Literal: "column1"}},
					Value: parser.FieldReference{Column: parser.Identifier{Literal: "notexist"}},
				},
			},
			WhereClause: parser.WhereClause{
				Filter: parser.Comparison{
					LHS:      parser.FieldReference{Column: parser.Identifier{Literal: "column1"}},
					RHS:      parser.NewIntegerValueFromString("2"),
					Operator: "=",
				},
			},
		},
		Error: "[L:- C:-] field notexist does not exist",
	},
	{
		Name: "Update Query Record Is Ambiguous Error",
		Query: parser.UpdateQuery{
			Tables: []parser.QueryExpression{
				parser.Table{Object: parser.Identifier{Literal: "t1"}},
			},
			SetList: []parser.UpdateSet{
				{
					Field: parser.FieldReference{Column: parser.Identifier{Literal: "column2"}},
					Value: parser.FieldReference{Column: parser.Identifier{Literal: "column4"}},
				},
			},
			FromClause: parser.FromClause{
				Tables: []parser.QueryExpression{
					parser.Table{Object: parser.Join{
						Table: parser.Table{
							Object: parser.Identifier{Literal: "table1"},
							Alias:  parser.Identifier{Literal: "t1"},
						},
						JoinTable: parser.Table{
							Object: parser.Identifier{Literal: "table2"},
							Alias:  parser.Identifier{Literal: "t2"},
						},
						JoinType: parser.Token{Token: parser.CROSS, Literal: "cross"},
					}},
				},
			},
		},
		Error: "[L:- C:-] value column4 to set in the field column2 is ambiguous",
	},
}

func TestUpdate(t *testing.T) {
	tf := cmd.GetFlags()
	tf.Repository = TestDir
	tf.Quiet = false

	filter := NewEmptyFilter()
	filter.TempViews = TemporaryViewScopes{
		ViewMap{
			"TMPVIEW": &View{
				Header: NewHeader("tmpview", []string{"column1", "column2"}),
				RecordSet: []Record{
					NewRecord([]value.Primary{
						value.NewString("1"),
						value.NewString("str1"),
					}),
					NewRecord([]value.Primary{
						value.NewString("2"),
						value.NewString("str2"),
					}),
				},
				FileInfo: &FileInfo{
					Path:        "tmpview",
					Delimiter:   ',',
					IsTemporary: true,
				},
			},
		},
	}

	for _, v := range updateTests {
		ReleaseResources()
		result, err := Update(v.Query, filter)
		if err != nil {
			if len(v.Error) < 1 {
				t.Errorf("%s: unexpected error %q", v.Name, err)
			} else if err.Error() != v.Error {
				t.Errorf("%s: error %q, want error %q", v.Name, err.Error(), v.Error)
			}
			continue
		}
		if 0 < len(v.Error) {
			t.Errorf("%s: no error, want error %q", v.Name, v.Error)
			continue
		}

		for _, v2 := range ViewCache {
			if v2.FileInfo.File != nil {
				if v2.FileInfo.Path != v2.FileInfo.File.Name() {
					t.Errorf("file pointer = %q, want %q for %q", v2.FileInfo.File.Name(), v2.FileInfo.Path, v.Name)
				}
				file.Close(v2.FileInfo.File)
				v2.FileInfo.File = nil
			}
		}

		if !reflect.DeepEqual(result, v.Result) {
			t.Errorf("%s: result = %v, want %v", v.Name, result, v.Result)
		}

		if v.ViewCache != nil {
			if !reflect.DeepEqual(ViewCache, v.ViewCache) {
				t.Errorf("%s: view cache = %v, want %v", v.Name, ViewCache, v.ViewCache)
			}
		}
		if v.TempViewList != nil {
			if !reflect.DeepEqual(filter.TempViews, v.TempViewList) {
				t.Errorf("%s: temporary views list = %v, want %v", v.Name, filter.TempViews, v.TempViewList)
			}
		}
	}
	ReleaseResources()
}

var deleteTests = []struct {
	Name         string
	Query        parser.DeleteQuery
	Result       []*View
	ViewCache    ViewMap
	TempViewList TemporaryViewScopes
	Error        string
}{
	{
		Name: "Delete Query",
		Query: parser.DeleteQuery{
			WithClause: parser.WithClause{
				With: "with",
				InlineTables: []parser.QueryExpression{
					parser.InlineTable{
						Name: parser.Identifier{Literal: "it"},
						Fields: []parser.QueryExpression{
							parser.Identifier{Literal: "c1"},
						},
						As: "as",
						Query: parser.SelectQuery{
							SelectEntity: parser.SelectEntity{
								SelectClause: parser.SelectClause{
									Select: "select",
									Fields: []parser.QueryExpression{
										parser.Field{Object: parser.NewIntegerValueFromString("2")},
									},
								},
							},
						},
					},
				},
			},
			FromClause: parser.FromClause{
				Tables: []parser.QueryExpression{
					parser.Table{
						Object: parser.Identifier{Literal: "table1"},
					},
				},
			},
			WhereClause: parser.WhereClause{
				Filter: parser.Comparison{
					LHS: parser.FieldReference{Column: parser.Identifier{Literal: "column1"}},
					RHS: parser.Subquery{
						Query: parser.SelectQuery{
							SelectEntity: parser.SelectEntity{
								SelectClause: parser.SelectClause{
									Select: "select",
									Fields: []parser.QueryExpression{
										parser.Field{Object: parser.FieldReference{View: parser.Identifier{Literal: "it"}, Column: parser.Identifier{Literal: "c1"}}},
									},
								},
								FromClause: parser.FromClause{
									Tables: []parser.QueryExpression{
										parser.Table{Object: parser.Identifier{Literal: "it"}},
									},
								},
							},
						},
					},
					Operator: "=",
				},
			},
		},
		Result: []*View{
			{
				FileInfo: &FileInfo{
					Path:      GetTestFilePath("table1.csv"),
					Delimiter: ',',
					NoHeader:  false,
					Encoding:  cmd.UTF8,
					LineBreak: cmd.LF,
				},
				Header: NewHeader("table1", []string{"column1", "column2"}),
				RecordSet: []Record{
					NewRecord([]value.Primary{
						value.NewString("1"),
						value.NewString("str1"),
					}),
					NewRecord([]value.Primary{
						value.NewString("3"),
						value.NewString("str3"),
					}),
				},
				ForUpdate:       true,
				OperatedRecords: 1,
			},
		},
		ViewCache: ViewMap{
			strings.ToUpper(GetTestFilePath("table1.csv")): &View{
				FileInfo: &FileInfo{
					Path:      GetTestFilePath("table1.csv"),
					Delimiter: ',',
					NoHeader:  false,
					Encoding:  cmd.UTF8,
					LineBreak: cmd.LF,
				},
				Header: NewHeader("table1", []string{"column1", "column2"}),
				RecordSet: []Record{
					NewRecord([]value.Primary{
						value.NewString("1"),
						value.NewString("str1"),
					}),
					NewRecord([]value.Primary{
						value.NewString("3"),
						value.NewString("str3"),
					}),
				},
				ForUpdate:       true,
				OperatedRecords: 1,
			},
		},
	},
	{
		Name: "Delete Query For Temporary View",
		Query: parser.DeleteQuery{
			FromClause: parser.FromClause{
				Tables: []parser.QueryExpression{
					parser.Table{
						Object: parser.Identifier{Literal: "tmpview"},
						Alias:  parser.Identifier{Literal: "t1"},
					},
				},
			},
			WhereClause: parser.WhereClause{
				Filter: parser.Comparison{
					LHS:      parser.FieldReference{Column: parser.Identifier{Literal: "column1"}},
					RHS:      parser.NewIntegerValueFromString("2"),
					Operator: "=",
				},
			},
		},
		Result: []*View{
			{
				FileInfo: &FileInfo{
					Path:        "tmpview",
					Delimiter:   ',',
					IsTemporary: true,
				},
				Header: NewHeader("tmpview", []string{"column1", "column2"}),
				RecordSet: []Record{
					NewRecord([]value.Primary{
						value.NewString("1"),
						value.NewString("str1"),
					}),
				},
				OperatedRecords: 1,
			},
		},
		TempViewList: TemporaryViewScopes{
			ViewMap{
				"TMPVIEW": &View{
					Header: NewHeader("tmpview", []string{"column1", "column2"}),
					RecordSet: []Record{
						NewRecord([]value.Primary{
							value.NewString("1"),
							value.NewString("str1"),
						}),
					},
					FileInfo: &FileInfo{
						Path:        "tmpview",
						Delimiter:   ',',
						IsTemporary: true,
					},
					OperatedRecords: 1,
				},
			},
		},
	},
	{
		Name: "Delete Query Multiple Table",
		Query: parser.DeleteQuery{
			Tables: []parser.QueryExpression{
				parser.Table{Object: parser.Identifier{Literal: "t1"}},
			},
			FromClause: parser.FromClause{
				Tables: []parser.QueryExpression{
					parser.Table{Object: parser.Join{
						Table: parser.Table{
							Object: parser.Identifier{Literal: "table1"},
							Alias:  parser.Identifier{Literal: "t1"},
						},
						JoinTable: parser.Table{
							Object: parser.Identifier{Literal: "table2"},
							Alias:  parser.Identifier{Literal: "t2"},
						},
						Condition: parser.JoinCondition{
							On: parser.Comparison{
								LHS:      parser.FieldReference{Column: parser.Identifier{Literal: "column1"}},
								RHS:      parser.FieldReference{Column: parser.Identifier{Literal: "column3"}},
								Operator: "=",
							},
						},
					}},
				},
			},
		},
		Result: []*View{
			{
				FileInfo: &FileInfo{
					Path:      GetTestFilePath("table1.csv"),
					Delimiter: ',',
					NoHeader:  false,
					Encoding:  cmd.UTF8,
					LineBreak: cmd.LF,
				},
				Header: NewHeader("table1", []string{"column1", "column2"}),
				RecordSet: []Record{
					NewRecord([]value.Primary{
						value.NewString("1"),
						value.NewString("str1"),
					}),
				},
				ForUpdate:       true,
				OperatedRecords: 2,
			},
		},
	},
	{
		Name: "Delete Query Tables Not Specified Error",
		Query: parser.DeleteQuery{
			FromClause: parser.FromClause{
				Tables: []parser.QueryExpression{
					parser.Table{Object: parser.Join{
						Table: parser.Table{
							Object: parser.Identifier{Literal: "table1"},
							Alias:  parser.Identifier{Literal: "t1"},
						},
						JoinTable: parser.Table{
							Object: parser.Identifier{Literal: "table2"},
							Alias:  parser.Identifier{Literal: "t2"},
						},
						Condition: parser.JoinCondition{
							On: parser.Comparison{
								LHS:      parser.FieldReference{Column: parser.Identifier{Literal: "column1"}},
								RHS:      parser.FieldReference{Column: parser.Identifier{Literal: "column3"}},
								Operator: "=",
							},
						},
					}},
				},
			},
		},
		Error: "[L:- C:-] tables to delete records are not specified",
	},
	{
		Name: "Delete Query File Does Not Exist Error",
		Query: parser.DeleteQuery{
			FromClause: parser.FromClause{
				Tables: []parser.QueryExpression{
					parser.Table{
						Object: parser.Identifier{Literal: "notexist"},
					},
				},
			},
			WhereClause: parser.WhereClause{
				Filter: parser.Comparison{
					LHS:      parser.FieldReference{Column: parser.Identifier{Literal: "column1"}},
					RHS:      parser.NewIntegerValueFromString("2"),
					Operator: "=",
				},
			},
		},
		Error: "[L:- C:-] file notexist does not exist",
	},
	{
		Name: "Delete Query Filter Error",
		Query: parser.DeleteQuery{
			FromClause: parser.FromClause{
				Tables: []parser.QueryExpression{
					parser.Table{
						Object: parser.Identifier{Literal: "table1"},
					},
				},
			},
			WhereClause: parser.WhereClause{
				Filter: parser.Comparison{
					LHS:      parser.FieldReference{Column: parser.Identifier{Literal: "column1"}},
					RHS:      parser.FieldReference{Column: parser.Identifier{Literal: "notexist"}},
					Operator: "=",
				},
			},
		},
		Error: "[L:- C:-] field notexist does not exist",
	},
	{
		Name: "Delete Query File Is Not Loaded Error",
		Query: parser.DeleteQuery{
			Tables: []parser.QueryExpression{
				parser.Table{Object: parser.Identifier{Literal: "notexist"}},
			},
			FromClause: parser.FromClause{
				Tables: []parser.QueryExpression{
					parser.Table{Object: parser.Join{
						Table: parser.Table{
							Object: parser.Identifier{Literal: "table1"},
							Alias:  parser.Identifier{Literal: "t1"},
						},
						JoinTable: parser.Table{
							Object: parser.Identifier{Literal: "table2"},
							Alias:  parser.Identifier{Literal: "t2"},
						},
						Condition: parser.JoinCondition{
							On: parser.Comparison{
								LHS:      parser.FieldReference{Column: parser.Identifier{Literal: "column1"}},
								RHS:      parser.FieldReference{Column: parser.Identifier{Literal: "column3"}},
								Operator: "=",
							},
						},
					}},
				},
			},
		},
		Error: "[L:- C:-] table notexist is not loaded",
	},
}

func TestDelete(t *testing.T) {
	tf := cmd.GetFlags()
	tf.Repository = TestDir
	tf.Quiet = false

	filter := NewEmptyFilter()
	filter.TempViews = TemporaryViewScopes{
		ViewMap{
			"TMPVIEW": &View{
				Header: NewHeader("tmpview", []string{"column1", "column2"}),
				RecordSet: []Record{
					NewRecord([]value.Primary{
						value.NewString("1"),
						value.NewString("str1"),
					}),
					NewRecord([]value.Primary{
						value.NewString("2"),
						value.NewString("str2"),
					}),
				},
				FileInfo: &FileInfo{
					Path:        "tmpview",
					Delimiter:   ',',
					IsTemporary: true,
				},
			},
		},
	}

	for _, v := range deleteTests {
		ReleaseResources()
		result, err := Delete(v.Query, filter)
		if err != nil {
			if len(v.Error) < 1 {
				t.Errorf("%s: unexpected error %q", v.Name, err)
			} else if err.Error() != v.Error {
				t.Errorf("%s: error %q, want error %q", v.Name, err.Error(), v.Error)
			}
			continue
		}
		if 0 < len(v.Error) {
			t.Errorf("%s: no error, want error %q", v.Name, v.Error)
			continue
		}

		for _, v2 := range ViewCache {
			if v2.FileInfo.File != nil {
				if v2.FileInfo.Path != v2.FileInfo.File.Name() {
					t.Errorf("file pointer = %q, want %q for %q", v2.FileInfo.File.Name(), v2.FileInfo.Path, v.Name)
				}
				file.Close(v2.FileInfo.File)
				v2.FileInfo.File = nil
			}
		}

		if !reflect.DeepEqual(result, v.Result) {
			t.Errorf("%s: result = %v, want %v", v.Name, result, v.Result)
		}

		if v.ViewCache != nil {
			if !reflect.DeepEqual(ViewCache, v.ViewCache) {
				t.Errorf("%s: view cache = %v, want %v", v.Name, ViewCache, v.ViewCache)
			}
		}
		if v.TempViewList != nil {
			if !reflect.DeepEqual(filter.TempViews, v.TempViewList) {
				t.Errorf("%s: temporary views list = %v, want %v", v.Name, filter.TempViews, v.TempViewList)
			}
		}
	}
	ReleaseResources()
}

var createTableTests = []struct {
	Name      string
	Query     parser.CreateTable
	Result    *View
	ViewCache ViewMap
	Error     string
}{
	{
		Name: "Create Table",
		Query: parser.CreateTable{
			Table: parser.Identifier{Literal: "create_table_1.csv"},
			Fields: []parser.QueryExpression{
				parser.Identifier{Literal: "column1"},
				parser.Identifier{Literal: "column2"},
			},
		},
		Result: &View{
			FileInfo: &FileInfo{
				Path:      GetTestFilePath("create_table_1.csv"),
				Delimiter: ',',
				NoHeader:  false,
				Encoding:  cmd.UTF8,
				LineBreak: cmd.LF,
			},
			Header:    NewHeader("create_table_1", []string{"column1", "column2"}),
			RecordSet: RecordSet{},
		},
		ViewCache: ViewMap{
			strings.ToUpper(GetTestFilePath("create_table_1.csv")): &View{
				FileInfo: &FileInfo{
					Path:      GetTestFilePath("create_table_1.csv"),
					Delimiter: ',',
					NoHeader:  false,
					Encoding:  cmd.UTF8,
					LineBreak: cmd.LF,
				},
				Header:    NewHeader("create_table_1", []string{"column1", "column2"}),
				RecordSet: RecordSet{},
			},
		},
	},
	{
		Name: "Create Table From Select Query",
		Query: parser.CreateTable{
			Table: parser.Identifier{Literal: "create_table_1.csv"},
			Fields: []parser.QueryExpression{
				parser.Identifier{Literal: "column1"},
				parser.Identifier{Literal: "column2"},
			},
			Query: parser.SelectQuery{
				SelectEntity: parser.SelectEntity{
					SelectClause: parser.SelectClause{
						Fields: []parser.QueryExpression{
							parser.Field{Object: parser.NewIntegerValueFromString("1")},
							parser.Field{Object: parser.NewIntegerValueFromString("2")},
						},
					},
				},
			},
		},
		Result: &View{
			FileInfo: &FileInfo{
				Path:      GetTestFilePath("create_table_1.csv"),
				Delimiter: ',',
				NoHeader:  false,
				Encoding:  cmd.UTF8,
				LineBreak: cmd.LF,
			},
			Header: NewHeader("create_table_1", []string{"column1", "column2"}),
			RecordSet: RecordSet{
				NewRecord([]value.Primary{
					value.NewInteger(1),
					value.NewInteger(2),
				}),
			},
		},
		ViewCache: ViewMap{
			strings.ToUpper(GetTestFilePath("create_table_1.csv")): &View{
				FileInfo: &FileInfo{
					Path:      GetTestFilePath("create_table_1.csv"),
					Delimiter: ',',
					NoHeader:  false,
					Encoding:  cmd.UTF8,
					LineBreak: cmd.LF,
				},
				Header: NewHeader("create_table_1", []string{"column1", "column2"}),
				RecordSet: RecordSet{
					NewRecord([]value.Primary{
						value.NewInteger(1),
						value.NewInteger(2),
					}),
				},
			},
		},
	},
	{
		Name: "Create Table File Already Exist Error",
		Query: parser.CreateTable{
			Table: parser.Identifier{Literal: "table1.csv"},
			Fields: []parser.QueryExpression{
				parser.Identifier{Literal: "column1"},
				parser.Identifier{Literal: "column2"},
			},
		},
		Error: "[L:- C:-] file table1.csv already exists",
	},
	{
		Name: "Create Table Field Duplicate Error",
		Query: parser.CreateTable{
			Table: parser.Identifier{Literal: "create_table_1.csv"},
			Fields: []parser.QueryExpression{
				parser.Identifier{Literal: "column1"},
				parser.Identifier{Literal: "column1"},
			},
		},
		Error: "[L:- C:-] field name column1 is a duplicate",
	},
	{
		Name: "Create Table Select Query Execution Error",
		Query: parser.CreateTable{
			Table: parser.Identifier{Literal: "create_table_1.csv"},
			Fields: []parser.QueryExpression{
				parser.Identifier{Literal: "column1"},
				parser.Identifier{Literal: "column2"},
			},
			Query: parser.SelectQuery{
				SelectEntity: parser.SelectEntity{
					SelectClause: parser.SelectClause{
						Fields: []parser.QueryExpression{
							parser.Field{Object: parser.FieldReference{Column: parser.Identifier{Literal: "notexist"}}},
							parser.Field{Object: parser.NewIntegerValueFromString("2")},
						},
					},
				},
			},
		},
		Error: "[L:- C:-] field notexist does not exist",
	},
	{
		Name: "Create Table From Select Query Field Length Not Match Error",
		Query: parser.CreateTable{
			Table: parser.Identifier{Literal: "create_table_1.csv"},
			Fields: []parser.QueryExpression{
				parser.Identifier{Literal: "column1"},
			},
			Query: parser.SelectQuery{
				SelectEntity: parser.SelectEntity{
					SelectClause: parser.SelectClause{
						Fields: []parser.QueryExpression{
							parser.Field{Object: parser.NewIntegerValueFromString("1")},
							parser.Field{Object: parser.NewIntegerValueFromString("2")},
						},
					},
				},
			},
		},
		Error: "[L:- C:-] select query should return exactly 1 field for table create_table_1.csv",
	},
	{
		Name: "Create Table From Select Query Field Name Duplicate Error",
		Query: parser.CreateTable{
			Table: parser.Identifier{Literal: "create_table_1.csv"},
			Fields: []parser.QueryExpression{
				parser.Identifier{Literal: "column1"},
				parser.Identifier{Literal: "column1"},
			},
			Query: parser.SelectQuery{
				SelectEntity: parser.SelectEntity{
					SelectClause: parser.SelectClause{
						Fields: []parser.QueryExpression{
							parser.Field{Object: parser.NewIntegerValueFromString("1")},
							parser.Field{Object: parser.NewIntegerValueFromString("2")},
						},
					},
				},
			},
		},
		Error: "[L:- C:-] field name column1 is a duplicate",
	},
}

func TestCreateTable(t *testing.T) {
	tf := cmd.GetFlags()
	tf.Repository = TestDir
	tf.Quiet = false

	for _, v := range createTableTests {
		ReleaseResources()
		result, err := CreateTable(v.Query, NewEmptyFilter())
		if err != nil {
			if len(v.Error) < 1 {
				t.Errorf("%s: unexpected error %q", v.Name, err)
			} else if err.Error() != v.Error {
				t.Errorf("%s: error %q, want error %q", v.Name, err.Error(), v.Error)
			}
			continue
		}
		if 0 < len(v.Error) {
			t.Errorf("%s: no error, want error %q", v.Name, v.Error)
			continue
		}
		if !reflect.DeepEqual(result, v.Result) {
			t.Errorf("%s: result = %v, want %v", v.Name, result, v.Result)
		}

		if v.ViewCache != nil {
			if !reflect.DeepEqual(ViewCache, v.ViewCache) {
				t.Errorf("%s: view cache = %v, want %v", v.Name, ViewCache, v.ViewCache)
			}
		}
	}
	ReleaseResources()
}

var addColumnsTests = []struct {
	Name         string
	Query        parser.AddColumns
	Result       *View
	ViewCache    ViewMap
	TempViewList TemporaryViewScopes
	Error        string
}{
	{
		Name: "Add Columns",
		Query: parser.AddColumns{
			Table: parser.Identifier{Literal: "table1.csv"},
			Columns: []parser.ColumnDefault{
				{
					Column: parser.Identifier{Literal: "column3"},
				},
				{
					Column: parser.Identifier{Literal: "column4"},
				},
			},
		},
		Result: &View{
			FileInfo: &FileInfo{
				Path:      GetTestFilePath("table1.csv"),
				Delimiter: ',',
				NoHeader:  false,
				Encoding:  cmd.UTF8,
				LineBreak: cmd.LF,
			},
			Header: NewHeader("table1", []string{"column1", "column2", "column3", "column4"}),
			RecordSet: []Record{
				NewRecord([]value.Primary{
					value.NewString("1"),
					value.NewString("str1"),
					value.NewNull(),
					value.NewNull(),
				}),
				NewRecord([]value.Primary{
					value.NewString("2"),
					value.NewString("str2"),
					value.NewNull(),
					value.NewNull(),
				}),
				NewRecord([]value.Primary{
					value.NewString("3"),
					value.NewString("str3"),
					value.NewNull(),
					value.NewNull(),
				}),
			},
			ForUpdate:      true,
			OperatedFields: 2,
		},
		ViewCache: ViewMap{
			strings.ToUpper(GetTestFilePath("table1.csv")): &View{
				FileInfo: &FileInfo{
					Path:      GetTestFilePath("table1.csv"),
					Delimiter: ',',
					NoHeader:  false,
					Encoding:  cmd.UTF8,
					LineBreak: cmd.LF,
				},
				Header: NewHeader("table1", []string{"column1", "column2", "column3", "column4"}),
				RecordSet: []Record{
					NewRecord([]value.Primary{
						value.NewString("1"),
						value.NewString("str1"),
						value.NewNull(),
						value.NewNull(),
					}),
					NewRecord([]value.Primary{
						value.NewString("2"),
						value.NewString("str2"),
						value.NewNull(),
						value.NewNull(),
					}),
					NewRecord([]value.Primary{
						value.NewString("3"),
						value.NewString("str3"),
						value.NewNull(),
						value.NewNull(),
					}),
				},
				ForUpdate:      true,
				OperatedFields: 2,
			},
		},
	},
	{
		Name: "Add Columns For Temporary View",
		Query: parser.AddColumns{
			Table: parser.Identifier{Literal: "tmpview"},
			Columns: []parser.ColumnDefault{
				{
					Column: parser.Identifier{Literal: "column3"},
				},
				{
					Column: parser.Identifier{Literal: "column4"},
				},
			},
		},
		Result: &View{
			FileInfo: &FileInfo{
				Path:        "tmpview",
				Delimiter:   ',',
				IsTemporary: true,
			},
			Header: NewHeader("tmpview", []string{"column1", "column2", "column3", "column4"}),
			RecordSet: []Record{
				NewRecord([]value.Primary{
					value.NewString("1"),
					value.NewString("str1"),
					value.NewNull(),
					value.NewNull(),
				}),
				NewRecord([]value.Primary{
					value.NewString("2"),
					value.NewString("str2"),
					value.NewNull(),
					value.NewNull(),
				}),
			},
			ForUpdate:      true,
			OperatedFields: 2,
		},
		TempViewList: TemporaryViewScopes{
			ViewMap{
				"TMPVIEW": &View{
					Header: NewHeader("tmpview", []string{"column1", "column2", "column3", "column4"}),
					RecordSet: []Record{
						NewRecord([]value.Primary{
							value.NewString("1"),
							value.NewString("str1"),
							value.NewNull(),
							value.NewNull(),
						}),
						NewRecord([]value.Primary{
							value.NewString("2"),
							value.NewString("str2"),
							value.NewNull(),
							value.NewNull(),
						}),
					},
					FileInfo: &FileInfo{
						Path:        "tmpview",
						Delimiter:   ',',
						IsTemporary: true,
					},
					ForUpdate:      true,
					OperatedFields: 2,
				},
			},
		},
	},
	{
		Name: "Add Columns First",
		Query: parser.AddColumns{
			Table: parser.Identifier{Literal: "table1.csv"},
			Columns: []parser.ColumnDefault{
				{
					Column: parser.Identifier{Literal: "column3"},
					Value:  parser.NewIntegerValueFromString("2"),
				},
				{
					Column: parser.Identifier{Literal: "column4"},
					Value:  parser.NewIntegerValueFromString("1"),
				},
			},
			Position: parser.ColumnPosition{
				Position: parser.Token{Token: parser.FIRST},
			},
		},
		Result: &View{
			FileInfo: &FileInfo{
				Path:      GetTestFilePath("table1.csv"),
				Delimiter: ',',
				NoHeader:  false,
				Encoding:  cmd.UTF8,
				LineBreak: cmd.LF,
			},
			Header: NewHeader("table1", []string{"column3", "column4", "column1", "column2"}),
			RecordSet: []Record{
				NewRecord([]value.Primary{
					value.NewInteger(2),
					value.NewInteger(1),
					value.NewString("1"),
					value.NewString("str1"),
				}),
				NewRecord([]value.Primary{
					value.NewInteger(2),
					value.NewInteger(1),
					value.NewString("2"),
					value.NewString("str2"),
				}),
				NewRecord([]value.Primary{
					value.NewInteger(2),
					value.NewInteger(1),
					value.NewString("3"),
					value.NewString("str3"),
				}),
			},
			ForUpdate:      true,
			OperatedFields: 2,
		},
	},
	{
		Name: "Add Columns After",
		Query: parser.AddColumns{
			Table: parser.Identifier{Literal: "table1.csv"},
			Columns: []parser.ColumnDefault{
				{
					Column: parser.Identifier{Literal: "column3"},
				},
				{
					Column: parser.Identifier{Literal: "column4"},
					Value:  parser.NewIntegerValueFromString("1"),
				},
			},
			Position: parser.ColumnPosition{
				Position: parser.Token{Token: parser.AFTER},
				Column:   parser.FieldReference{Column: parser.Identifier{Literal: "column1"}},
			},
		},
		Result: &View{
			FileInfo: &FileInfo{
				Path:      GetTestFilePath("table1.csv"),
				Delimiter: ',',
				NoHeader:  false,
				Encoding:  cmd.UTF8,
				LineBreak: cmd.LF,
			},
			Header: NewHeader("table1", []string{"column1", "column3", "column4", "column2"}),
			RecordSet: []Record{
				NewRecord([]value.Primary{
					value.NewString("1"),
					value.NewNull(),
					value.NewInteger(1),
					value.NewString("str1"),
				}),
				NewRecord([]value.Primary{
					value.NewString("2"),
					value.NewNull(),
					value.NewInteger(1),
					value.NewString("str2"),
				}),
				NewRecord([]value.Primary{
					value.NewString("3"),
					value.NewNull(),
					value.NewInteger(1),
					value.NewString("str3"),
				}),
			},
			ForUpdate:      true,
			OperatedFields: 2,
		},
	},
	{
		Name: "Add Columns Before",
		Query: parser.AddColumns{
			Table: parser.Identifier{Literal: "table1.csv"},
			Columns: []parser.ColumnDefault{
				{
					Column: parser.Identifier{Literal: "column3"},
				},
				{
					Column: parser.Identifier{Literal: "column4"},
					Value:  parser.NewIntegerValueFromString("1"),
				},
			},
			Position: parser.ColumnPosition{
				Position: parser.Token{Token: parser.BEFORE},
				Column:   parser.ColumnNumber{View: parser.Identifier{Literal: "table1"}, Number: value.NewInteger(2)},
			},
		},
		Result: &View{
			FileInfo: &FileInfo{
				Path:      GetTestFilePath("table1.csv"),
				Delimiter: ',',
				NoHeader:  false,
				Encoding:  cmd.UTF8,
				LineBreak: cmd.LF,
			},
			Header: NewHeader("table1", []string{"column1", "column3", "column4", "column2"}),
			RecordSet: []Record{
				NewRecord([]value.Primary{
					value.NewString("1"),
					value.NewNull(),
					value.NewInteger(1),
					value.NewString("str1"),
				}),
				NewRecord([]value.Primary{
					value.NewString("2"),
					value.NewNull(),
					value.NewInteger(1),
					value.NewString("str2"),
				}),
				NewRecord([]value.Primary{
					value.NewString("3"),
					value.NewNull(),
					value.NewInteger(1),
					value.NewString("str3"),
				}),
			},
			ForUpdate:      true,
			OperatedFields: 2,
		},
	},
	{
		Name: "Add Columns Load Error",
		Query: parser.AddColumns{
			Table: parser.Identifier{Literal: "notexist"},
			Columns: []parser.ColumnDefault{
				{
					Column: parser.Identifier{Literal: "column3"},
				},
				{
					Column: parser.Identifier{Literal: "column4"},
				},
			},
		},
		Error: "[L:- C:-] file notexist does not exist",
	},
	{
		Name: "Add Columns Position Column Does Not Exist Error",
		Query: parser.AddColumns{
			Table: parser.Identifier{Literal: "table1.csv"},
			Columns: []parser.ColumnDefault{
				{
					Column: parser.Identifier{Literal: "column3"},
				},
				{
					Column: parser.Identifier{Literal: "column2"},
					Value:  parser.NewIntegerValueFromString("1"),
				},
			},
			Position: parser.ColumnPosition{
				Position: parser.Token{Token: parser.BEFORE},
				Column:   parser.FieldReference{Column: parser.Identifier{Literal: "notexist"}},
			},
		},
		Error: "[L:- C:-] field notexist does not exist",
	},
	{
		Name: "Add Columns Field Duplicate Error",
		Query: parser.AddColumns{
			Table: parser.Identifier{Literal: "table1.csv"},
			Columns: []parser.ColumnDefault{
				{
					Column: parser.Identifier{Literal: "column3"},
				},
				{
					Column: parser.Identifier{Literal: "column1"},
					Value:  parser.NewIntegerValueFromString("1"),
				},
			},
		},
		Error: "[L:- C:-] field name column1 is a duplicate",
	},
	{
		Name: "Add Columns Default Value Error",
		Query: parser.AddColumns{
			Table: parser.Identifier{Literal: "table1.csv"},
			Columns: []parser.ColumnDefault{
				{
					Column: parser.Identifier{Literal: "column3"},
				},
				{
					Column: parser.Identifier{Literal: "column4"},
					Value:  parser.FieldReference{Column: parser.Identifier{Literal: "notexist"}},
				},
			},
		},
		Error: "[L:- C:-] field notexist does not exist",
	},
}

func TestAddColumns(t *testing.T) {
	tf := cmd.GetFlags()
	tf.Repository = TestDir
	tf.Quiet = false

	filter := NewEmptyFilter()
	filter.TempViews = TemporaryViewScopes{
		ViewMap{
			"TMPVIEW": &View{
				Header: NewHeader("tmpview", []string{"column1", "column2"}),
				RecordSet: []Record{
					NewRecord([]value.Primary{
						value.NewString("1"),
						value.NewString("str1"),
					}),
					NewRecord([]value.Primary{
						value.NewString("2"),
						value.NewString("str2"),
					}),
				},
				FileInfo: &FileInfo{
					Path:        "tmpview",
					Delimiter:   ',',
					IsTemporary: true,
				},
			},
		},
	}
	for _, v := range addColumnsTests {
		ReleaseResources()
		result, err := AddColumns(v.Query, filter)
		if err != nil {
			if len(v.Error) < 1 {
				t.Errorf("%s: unexpected error %q", v.Name, err)
			} else if err.Error() != v.Error {
				t.Errorf("%s: error %q, want error %q", v.Name, err.Error(), v.Error)
			}
			continue
		}
		if 0 < len(v.Error) {
			t.Errorf("%s: no error, want error %q", v.Name, v.Error)
			continue
		}

		for _, v2 := range ViewCache {
			if v2.FileInfo.File != nil {
				if v2.FileInfo.Path != v2.FileInfo.File.Name() {
					t.Errorf("file pointer = %q, want %q for %q", v2.FileInfo.File.Name(), v2.FileInfo.Path, v.Name)
				}
				file.Close(v2.FileInfo.File)
				v2.FileInfo.File = nil
			}
		}

		if !reflect.DeepEqual(result, v.Result) {
			t.Errorf("%s: result = %v, want %v", v.Name, result, v.Result)
		}

		if v.ViewCache != nil {
			if !reflect.DeepEqual(ViewCache, v.ViewCache) {
				t.Errorf("%s: view cache = %v, want %v", v.Name, ViewCache, v.ViewCache)
			}
		}
		if v.TempViewList != nil {
			if !reflect.DeepEqual(filter.TempViews, v.TempViewList) {
				t.Errorf("%s: temporary views list = %v, want %v", v.Name, filter.TempViews, v.TempViewList)
			}
		}
	}
	ReleaseResources()
}

var dropColumnsTests = []struct {
	Name         string
	Query        parser.DropColumns
	Result       *View
	ViewCache    ViewMap
	TempViewList TemporaryViewScopes
	Error        string
}{
	{
		Name: "Drop Columns",
		Query: parser.DropColumns{
			Table: parser.Identifier{Literal: "table1"},
			Columns: []parser.QueryExpression{
				parser.FieldReference{Column: parser.Identifier{Literal: "column2"}},
			},
		},
		Result: &View{
			FileInfo: &FileInfo{
				Path:      GetTestFilePath("table1.csv"),
				Delimiter: ',',
				NoHeader:  false,
				Encoding:  cmd.UTF8,
				LineBreak: cmd.LF,
			},
			Header: NewHeader("table1", []string{"column1"}),
			RecordSet: []Record{
				NewRecord([]value.Primary{
					value.NewString("1"),
				}),
				NewRecord([]value.Primary{
					value.NewString("2"),
				}),
				NewRecord([]value.Primary{
					value.NewString("3"),
				}),
			},
			ForUpdate:      true,
			OperatedFields: 1,
		},
		ViewCache: ViewMap{
			strings.ToUpper(GetTestFilePath("table1.csv")): &View{
				FileInfo: &FileInfo{
					Path:      GetTestFilePath("table1.csv"),
					Delimiter: ',',
					NoHeader:  false,
					Encoding:  cmd.UTF8,
					LineBreak: cmd.LF,
				},
				Header: NewHeader("table1", []string{"column1"}),
				RecordSet: []Record{
					NewRecord([]value.Primary{
						value.NewString("1"),
					}),
					NewRecord([]value.Primary{
						value.NewString("2"),
					}),
					NewRecord([]value.Primary{
						value.NewString("3"),
					}),
				},
				ForUpdate:      true,
				OperatedFields: 1,
			},
		},
	},
	{
		Name: "Drop Columns For Temporary View",
		Query: parser.DropColumns{
			Table: parser.Identifier{Literal: "tmpview"},
			Columns: []parser.QueryExpression{
				parser.ColumnNumber{View: parser.Identifier{Literal: "tmpview"}, Number: value.NewInteger(2)},
			},
		},
		Result: &View{
			FileInfo: &FileInfo{
				Path:        "tmpview",
				Delimiter:   ',',
				IsTemporary: true,
			},
			Header: NewHeader("tmpview", []string{"column1"}),
			RecordSet: []Record{
				NewRecord([]value.Primary{
					value.NewString("1"),
				}),
				NewRecord([]value.Primary{
					value.NewString("2"),
				}),
			},
			ForUpdate:      true,
			OperatedFields: 1,
		},
		TempViewList: TemporaryViewScopes{
			ViewMap{
				"TMPVIEW": &View{
					Header: NewHeader("tmpview", []string{"column1"}),
					RecordSet: []Record{
						NewRecord([]value.Primary{
							value.NewString("1"),
						}),
						NewRecord([]value.Primary{
							value.NewString("2"),
						}),
					},
					FileInfo: &FileInfo{
						Path:        "tmpview",
						Delimiter:   ',',
						IsTemporary: true,
					},
					ForUpdate:      true,
					OperatedFields: 1,
				},
			},
		},
	},
	{
		Name: "Drop Columns Load Error",
		Query: parser.DropColumns{
			Table: parser.Identifier{Literal: "notexist"},
			Columns: []parser.QueryExpression{
				parser.FieldReference{Column: parser.Identifier{Literal: "column2"}},
			},
		},
		Error: "[L:- C:-] file notexist does not exist",
	},
	{
		Name: "Drop Columns Field Does Not Exist Error",
		Query: parser.DropColumns{
			Table: parser.Identifier{Literal: "table1"},
			Columns: []parser.QueryExpression{
				parser.FieldReference{Column: parser.Identifier{Literal: "notexist"}},
			},
		},
		Error: "[L:- C:-] field notexist does not exist",
	},
}

func TestDropColumns(t *testing.T) {
	tf := cmd.GetFlags()
	tf.Repository = TestDir
	tf.Quiet = false

	filter := NewEmptyFilter()
	filter.TempViews = TemporaryViewScopes{
		ViewMap{
			"TMPVIEW": &View{
				Header: NewHeader("tmpview", []string{"column1", "column2"}),
				RecordSet: []Record{
					NewRecord([]value.Primary{
						value.NewString("1"),
						value.NewString("str1"),
					}),
					NewRecord([]value.Primary{
						value.NewString("2"),
						value.NewString("str2"),
					}),
				},
				FileInfo: &FileInfo{
					Path:        "tmpview",
					Delimiter:   ',',
					IsTemporary: true,
				},
			},
		},
	}

	for _, v := range dropColumnsTests {
		ReleaseResources()
		result, err := DropColumns(v.Query, filter)
		if err != nil {
			if len(v.Error) < 1 {
				t.Errorf("%s: unexpected error %q", v.Name, err)
			} else if err.Error() != v.Error {
				t.Errorf("%s: error %q, want error %q", v.Name, err.Error(), v.Error)
			}
			continue
		}
		if 0 < len(v.Error) {
			t.Errorf("%s: no error, want error %q", v.Name, v.Error)
			continue
		}

		for _, v2 := range ViewCache {
			if v2.FileInfo.File != nil {
				if v2.FileInfo.Path != v2.FileInfo.File.Name() {
					t.Errorf("file pointer = %q, want %q for %q", v2.FileInfo.File.Name(), v2.FileInfo.Path, v.Name)
				}
				file.Close(v2.FileInfo.File)
				v2.FileInfo.File = nil
			}
		}

		if !reflect.DeepEqual(result, v.Result) {
			t.Errorf("%s: result = %v, want %v", v.Name, result, v.Result)
		}

		if v.ViewCache != nil {
			if !reflect.DeepEqual(ViewCache, v.ViewCache) {
				t.Errorf("%s: view cache = %v, want %v", v.Name, ViewCache, v.ViewCache)
			}
		}
		if v.TempViewList != nil {
			if !reflect.DeepEqual(filter.TempViews, v.TempViewList) {
				t.Errorf("%s: temporary views list = %v, want %v", v.Name, filter.TempViews, v.TempViewList)
			}
		}
	}
	ReleaseResources()
}

var renameColumnTests = []struct {
	Name         string
	Query        parser.RenameColumn
	Result       *View
	ViewCache    ViewMap
	TempViewList TemporaryViewScopes
	Error        string
}{
	{
		Name: "Rename Column",
		Query: parser.RenameColumn{
			Table: parser.Identifier{Literal: "table1"},
			Old:   parser.FieldReference{Column: parser.Identifier{Literal: "column2"}},
			New:   parser.Identifier{Literal: "newcolumn"},
		},
		Result: &View{
			FileInfo: &FileInfo{
				Path:      GetTestFilePath("table1.csv"),
				Delimiter: ',',
				NoHeader:  false,
				Encoding:  cmd.UTF8,
				LineBreak: cmd.LF,
			},
			Header: NewHeader("table1", []string{"column1", "newcolumn"}),
			RecordSet: []Record{
				NewRecord([]value.Primary{
					value.NewString("1"),
					value.NewString("str1"),
				}),
				NewRecord([]value.Primary{
					value.NewString("2"),
					value.NewString("str2"),
				}),
				NewRecord([]value.Primary{
					value.NewString("3"),
					value.NewString("str3"),
				}),
			},
			ForUpdate:      true,
			OperatedFields: 1,
		},
		ViewCache: ViewMap{
			strings.ToUpper(GetTestFilePath("table1.csv")): &View{
				FileInfo: &FileInfo{
					Path:      GetTestFilePath("table1.csv"),
					Delimiter: ',',
					NoHeader:  false,
					Encoding:  cmd.UTF8,
					LineBreak: cmd.LF,
				},
				Header: NewHeader("table1", []string{"column1", "newcolumn"}),
				RecordSet: []Record{
					NewRecord([]value.Primary{
						value.NewString("1"),
						value.NewString("str1"),
					}),
					NewRecord([]value.Primary{
						value.NewString("2"),
						value.NewString("str2"),
					}),
					NewRecord([]value.Primary{
						value.NewString("3"),
						value.NewString("str3"),
					}),
				},
				ForUpdate:      true,
				OperatedFields: 1,
			},
		},
	},
	{
		Name: "Rename Column For Temporary View",
		Query: parser.RenameColumn{
			Table: parser.Identifier{Literal: "tmpview"},
			Old:   parser.ColumnNumber{View: parser.Identifier{Literal: "tmpview"}, Number: value.NewInteger(2)},
			New:   parser.Identifier{Literal: "newcolumn"},
		},
		Result: &View{
			FileInfo: &FileInfo{
				Path:        "tmpview",
				Delimiter:   ',',
				IsTemporary: true,
			},
			Header: NewHeader("tmpview", []string{"column1", "newcolumn"}),
			RecordSet: []Record{
				NewRecord([]value.Primary{
					value.NewString("1"),
					value.NewString("str1"),
				}),
				NewRecord([]value.Primary{
					value.NewString("2"),
					value.NewString("str2"),
				}),
			},
			ForUpdate:      true,
			OperatedFields: 1,
		},
		TempViewList: TemporaryViewScopes{
			ViewMap{
				"TMPVIEW": &View{
					Header: NewHeader("tmpview", []string{"column1", "newcolumn"}),
					RecordSet: []Record{
						NewRecord([]value.Primary{
							value.NewString("1"),
							value.NewString("str1"),
						}),
						NewRecord([]value.Primary{
							value.NewString("2"),
							value.NewString("str2"),
						}),
					},
					FileInfo: &FileInfo{
						Path:        "tmpview",
						Delimiter:   ',',
						IsTemporary: true,
					},
					ForUpdate:      true,
					OperatedFields: 1,
				},
			},
		},
	},
	{
		Name: "Rename Column Load Error",
		Query: parser.RenameColumn{
			Table: parser.Identifier{Literal: "notexist"},
			Old:   parser.FieldReference{Column: parser.Identifier{Literal: "column2"}},
			New:   parser.Identifier{Literal: "newcolumn"},
		},
		Error: "[L:- C:-] file notexist does not exist",
	},
	{
		Name: "Rename Column Field Duplicate Error",
		Query: parser.RenameColumn{
			Table: parser.Identifier{Literal: "table1"},
			Old:   parser.FieldReference{Column: parser.Identifier{Literal: "column2"}},
			New:   parser.Identifier{Literal: "column1"},
		},
		Error: "[L:- C:-] field name column1 is a duplicate",
	},
	{
		Name: "Rename Column Field Does Not Exist Error",
		Query: parser.RenameColumn{
			Table: parser.Identifier{Literal: "table1"},
			Old:   parser.FieldReference{Column: parser.Identifier{Literal: "notexist"}},
			New:   parser.Identifier{Literal: "newcolumn"},
		},
		Error: "[L:- C:-] field notexist does not exist",
	},
}

func TestRenameColumn(t *testing.T) {
	tf := cmd.GetFlags()
	tf.Repository = TestDir
	tf.Quiet = false

	filter := NewEmptyFilter()
	filter.TempViews = TemporaryViewScopes{
		ViewMap{
			"TMPVIEW": &View{
				Header: NewHeader("tmpview", []string{"column1", "column2"}),
				RecordSet: []Record{
					NewRecord([]value.Primary{
						value.NewString("1"),
						value.NewString("str1"),
					}),
					NewRecord([]value.Primary{
						value.NewString("2"),
						value.NewString("str2"),
					}),
				},
				FileInfo: &FileInfo{
					Path:        "tmpview",
					Delimiter:   ',',
					IsTemporary: true,
				},
			},
		},
	}

	for _, v := range renameColumnTests {
		ReleaseResources()
		result, err := RenameColumn(v.Query, filter)
		if err != nil {
			if len(v.Error) < 1 {
				t.Errorf("%s: unexpected error %q", v.Name, err)
			} else if err.Error() != v.Error {
				t.Errorf("%s: error %q, want error %q", v.Name, err.Error(), v.Error)
			}
			continue
		}
		if 0 < len(v.Error) {
			t.Errorf("%s: no error, want error %q", v.Name, v.Error)
			continue
		}

		for _, v2 := range ViewCache {
			if v2.FileInfo.File != nil {
				if v2.FileInfo.Path != v2.FileInfo.File.Name() {
					t.Errorf("file pointer = %q, want %q for %q", v2.FileInfo.File.Name(), v2.FileInfo.Path, v.Name)
				}
				file.Close(v2.FileInfo.File)
				v2.FileInfo.File = nil
			}
		}

		if !reflect.DeepEqual(result, v.Result) {
			t.Errorf("%s: result = %v, want %v", v.Name, result, v.Result)
		}

		if v.ViewCache != nil {
			if !reflect.DeepEqual(ViewCache, v.ViewCache) {
				t.Errorf("%s: view cache = %v, want %v", v.Name, ViewCache, v.ViewCache)
			}
		}
		if v.TempViewList != nil {
			if !reflect.DeepEqual(filter.TempViews, v.TempViewList) {
				t.Errorf("%s: temporary views list = %v, want %v", v.Name, filter.TempViews, v.TempViewList)
			}
		}
	}
	ReleaseResources()
}

func TestCommit(t *testing.T) {
	cmd.SetQuiet(false)

	fp, _ := file.OpenToUpdate(GetTestFilePath("updated_file_1.csv"))

	ViewCache = ViewMap{
		strings.ToUpper(GetTestFilePath("created_file.csv")): &View{
			Header:    NewHeader("created_file", []string{"column1", "column2"}),
			RecordSet: RecordSet{},
			FileInfo: &FileInfo{
				Path: GetTestFilePath("created_file.csv"),
			},
		},
		strings.ToUpper(GetTestFilePath("updated_file_1.csv")): &View{
			Header: NewHeader("table1", []string{"column1", "column2"}),
			RecordSet: []Record{
				NewRecord([]value.Primary{
					value.NewString("1"),
					value.NewString("str1"),
				}),
				NewRecord([]value.Primary{
					value.NewString("update1"),
					value.NewString("update2"),
				}),
				NewRecord([]value.Primary{
					value.NewString("3"),
					value.NewString("str3"),
				}),
			},
			FileInfo: &FileInfo{
				Path: GetTestFilePath("updated_file_1.csv"),
				File: fp,
			},
		},
	}

	Results = []Result{
		{
			Type: CREATE_TABLE,
			FileInfo: &FileInfo{
				Path: GetTestFilePath("created_file.csv"),
			},
		},
		{
			Type: UPDATE,
			FileInfo: &FileInfo{
				Path: GetTestFilePath("updated_file_1.csv"),
				File: fp,
			},
			OperatedCount: 1,
		},
		{
			Type: UPDATE,
			FileInfo: &FileInfo{
				Path: GetTestFilePath("updated_file_2.csv"),
			},
		},
	}
	expect := fmt.Sprintf("Commit: file %q is created.\nCommit: file %q is updated.\n", GetTestFilePath("created_file.csv"), GetTestFilePath("updated_file_1.csv"))

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	Commit(parser.TransactionControl{Token: parser.COMMIT}, NewEmptyFilter())

	w.Close()
	os.Stdout = oldStdout
	log, _ := ioutil.ReadAll(r)

	if string(log) != expect {
		t.Errorf("Commit: log = %q, want %q", string(log), expect)
	}
}

func TestRollback(t *testing.T) {
	cmd.SetQuiet(false)

	Results = []Result{
		{
			Type: CREATE_TABLE,
			FileInfo: &FileInfo{
				Path: "created_file.csv",
			},
		},
		{
			Type: UPDATE,
			FileInfo: &FileInfo{
				Path: "updated_file_1.csv",
			},
			OperatedCount: 1,
		},
		{
			Type: UPDATE,
			FileInfo: &FileInfo{
				Path: "updated_file_2.csv",
			},
		},
	}
	expect := "Rollback: file \"created_file.csv\" is deleted.\nRollback: file \"updated_file_1.csv\" is restored.\n"

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	Rollback(NewEmptyFilter())

	w.Close()
	os.Stdout = oldStdout
	log, _ := ioutil.ReadAll(r)

	if string(log) != expect {
		t.Errorf("Rollback: log = %q, want %q", string(log), expect)
	}
}
