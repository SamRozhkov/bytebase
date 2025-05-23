package mysql

import (
	"sort"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_MYSQL, validateQuery)
	base.RegisterQueryValidator(storepb.Engine_MARIADB, validateQuery)
	base.RegisterQueryValidator(storepb.Engine_OCEANBASE, validateQuery)
	base.RegisterExtractResourceListFunc(storepb.Engine_MYSQL, ExtractResourceList)
	base.RegisterExtractResourceListFunc(storepb.Engine_MARIADB, ExtractResourceList)
	base.RegisterExtractResourceListFunc(storepb.Engine_OCEANBASE, ExtractResourceList)
	base.RegisterExtractResourceListFunc(storepb.Engine_STARROCKS, ExtractResourceList)
}

// validateQuery validates the SQL statement for SQL editor.
// We validate the statement by following steps:
// 1. Remove all quoted text(quoted identifier, string literal) and comments from the statement.
// 2. Use regexp to check if the statement is a normal SELECT statement and EXPLAIN statement.
// 3. For CTE, use regexp to check if the statement has UPDATE, DELETE and INSERT statements.
func validateQuery(statement string) (bool, bool, error) {
	trees, err := ParseMySQL(statement)
	if err != nil {
		return false, false, err
	}
	hasExecute := false
	readOnly := true
	for _, item := range trees {
		l := &queryValidateListener{
			valid:      true,
			hasExecute: false,
		}
		antlr.ParseTreeWalkerDefault.Walk(l, item.Tree)
		if !l.valid {
			return false, false, nil
		}
		if l.explainAnalyze {
			readOnly = false
		}
		if l.hasExecute {
			hasExecute = true
		}
	}
	return readOnly, !hasExecute, nil
}

type queryValidateListener struct {
	*parser.BaseMySQLParserListener

	valid          bool
	explainAnalyze bool
	hasExecute     bool
}

// EnterQuery is called when production query is entered.
func (l *queryValidateListener) EnterQuery(ctx *parser.QueryContext) {
	if !l.valid {
		return
	}
	if ctx.BeginWork() != nil {
		l.valid = false
	}
}

// EnterSimpleStatement is called when production simpleStatement is entered.
func (l *queryValidateListener) EnterSimpleStatement(ctx *parser.SimpleStatementContext) {
	if !l.valid {
		return
	}
	if ctx.SetStatement() != nil {
		l.hasExecute = true
	}
	if ctx.SelectStatement() == nil && ctx.UtilityStatement() == nil && ctx.SetStatement() == nil && ctx.ShowStatement() == nil {
		l.valid = false
	}
}

// EnterUtilityStatement is called when production utilityStatement is entered.
func (l *queryValidateListener) EnterUtilityStatement(ctx *parser.UtilityStatementContext) {
	if !l.valid {
		return
	}
	if ctx.ExplainStatement() == nil && ctx.DescribeStatement() == nil {
		l.valid = false
		return
	}
	if ctx.ExplainStatement() != nil {
		if ctx.ExplainStatement().ANALYZE_SYMBOL() != nil {
			l.explainAnalyze = true
			return
		}
	}
}

func ExtractResourceList(currentDatabase string, _, statement string) ([]base.SchemaResource, error) {
	treeList, err := ParseMySQL(statement)
	if err != nil {
		return nil, err
	}

	l := &resourceExtractListener{
		currentDatabase: currentDatabase,
		resourceMap:     make(map[string]base.SchemaResource),
	}

	var result []base.SchemaResource
	for _, tree := range treeList {
		if tree == nil {
			continue
		}
		antlr.ParseTreeWalkerDefault.Walk(l, tree.Tree)
	}
	for _, resource := range l.resourceMap {
		result = append(result, resource)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].String() < result[j].String()
	})

	return result, nil
}

type resourceExtractListener struct {
	*parser.BaseMySQLParserListener

	currentDatabase string
	resourceMap     map[string]base.SchemaResource
}

// EnterTableRef is called when production tableRef is entered.
func (l *resourceExtractListener) EnterTableRef(ctx *parser.TableRefContext) {
	resource := base.SchemaResource{Database: l.currentDatabase}
	if ctx.DotIdentifier() != nil {
		resource.Table = NormalizeMySQLIdentifier(ctx.DotIdentifier().Identifier())
	}
	db, table := normalizeMySQLQualifiedIdentifier(ctx.QualifiedIdentifier())
	if db != "" {
		resource.Database = db
	}
	resource.Table = table
	l.resourceMap[resource.String()] = resource
}
