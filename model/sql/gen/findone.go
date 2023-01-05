package gen

import (
	"fmt"
	"github.com/zeromicro/go-zero/tools/goctl/model/sql/template"
	"github.com/zeromicro/go-zero/tools/goctl/util"
	"github.com/zeromicro/go-zero/tools/goctl/util/pathx"
	"github.com/zeromicro/go-zero/tools/goctl/util/stringx"
	"regexp"
	"strings"
)

var (
	// https://github.com/golang/lint/blob/master/lint.go#L770
	commonInitialisms         = []string{"API", "ASCII", "CPU", "CSS", "DNS", "EOF", "GUID", "HTML", "HTTP", "HTTPS", "ID", "IP", "JSON", "LHS", "QPS", "RAM", "RHS", "RPC", "SLA", "SMTP", "SSH", "TLS", "TTL", "UID", "UI", "UUID", "URI", "URL", "UTF8", "VM", "XML", "XSRF", "XSS"}
	commonInitialismsReplacer *strings.Replacer
)

func init() {
	commonInitialismsForReplacer := make([]string, 0, len(commonInitialisms))
	for _, initialism := range commonInitialisms {
		commonInitialismsForReplacer = append(commonInitialismsForReplacer, initialism, strings.Title(strings.ToLower(initialism)))
	}
	commonInitialismsReplacer = strings.NewReplacer(commonInitialismsForReplacer...)
}
func toSchemaName(name string) string {
	result := strings.ReplaceAll(strings.Title(strings.ReplaceAll(name, "_", " ")), " ", "")
	for _, initialism := range commonInitialisms {
		result = regexp.MustCompile(strings.Title(strings.ToLower(initialism))+"([A-Z]|$|_)").ReplaceAllString(result, initialism+"$1")
	}
	return result
}

// CheckDataType 检测类型
func CheckDataType(s string) string {
	if s == "int64" || s == "int32" || s == "int" {
		return " != 0"
	}

	if s == "string" {
		return ` != ""`
	}

	if s == "time.Time" {
		return `.IsZero() == false`
	}

	return ""
}
func genIF(table Table) string {
	camel := table.Name.ToCamel()
	fields := table.Fields

	tableName := stringx.From(camel).Untitle()
	tableQb := stringx.From(camel).Untitle() + "Qb"
	//ifFields := fmt.Sprintf("%s := %s.WithContext(ctx)\n", tableQb, tableName)
	ifFields := ""

	for _, field := range fields {
		name := toSchemaName(field.Name.Source())
		if name == "DeletedAt" {
			continue
		}
		check := CheckDataType(field.DataType)

		//if field.Name.ToCamel() == table.PrimaryKey.Name.ToCamel() {
		//	name = stringx.From(name).Upper()
		//	ifFields = ifFields + fmt.Sprintf("if data.%s%s {\n", name, check)
		//} else {
		//	ifFields = ifFields + fmt.Sprintf("if data.%s%s {\n", toDBName(name), check)
		//}

		ifFields = ifFields + fmt.Sprintf("if data.%s%s {\n", name, check)
		ifFields = ifFields + fmt.Sprintf("%s = %s.Where(%s.%s.Eq(data.%s))\n}\n", tableQb, tableQb, tableName, name, name)
	}
	return ifFields
}
func genFindOne(table Table, withCache, postgreSql bool) (string, string, error) {
	camel := table.Name.ToCamel()
	text, err := pathx.LoadTemplate(category, findOneTemplateFile, template.FindOne)
	if err != nil {
		return "", "", err
	}

	ifFields := genIF(table)

	output, err := util.With("findOne").
		Parse(text).
		Execute(map[string]interface{}{
			"withCache":                 withCache,
			"upperStartCamelObject":     camel,
			"ifFields":                  ifFields,
			"lowerStartCamelObject":     stringx.From(camel).Untitle(),
			"originalPrimaryKey":        wrapWithRawString(table.PrimaryKey.Name.Source(), postgreSql),
			"lowerStartCamelPrimaryKey": util.EscapeGolangKeyword(stringx.From(table.PrimaryKey.Name.ToCamel()).Untitle()),
			"upperStartCamelPrimaryKey": util.EscapeGolangKeyword(stringx.From(table.PrimaryKey.Name.ToCamel()).Title()),
			"UpperStartCamelPrimaryKey": toSchemaName(table.PrimaryKey.Name.Source()),
			"dataType":                  table.PrimaryKey.DataType,
			"cacheKey":                  table.PrimaryCacheKey.KeyExpression,
			"cacheKeyVariable":          table.PrimaryCacheKey.KeyLeft,
			"postgreSql":                postgreSql,
			"data":                      table,
		})
	if err != nil {
		return "", "", err
	}

	text, err = pathx.LoadTemplate(category, findOneMethodTemplateFile, template.FindOneMethod)
	if err != nil {
		return "", "", err
	}

	findOneMethod, err := util.With("findOneMethod").
		Parse(text).
		Execute(map[string]interface{}{
			"upperStartCamelObject":     camel,
			"lowerStartCamelObject":     stringx.From(camel).Untitle(),
			"lowerStartCamelPrimaryKey": util.EscapeGolangKeyword(stringx.From(table.PrimaryKey.Name.ToCamel()).Untitle()),
			"upperStartCamelPrimaryKey": util.EscapeGolangKeyword(stringx.From(table.PrimaryKey.Name.ToCamel()).Title()),
			"UpperStartCamelPrimaryKey": util.EscapeGolangKeyword(stringx.From(table.PrimaryKey.Name.ToCamel()).Upper()),
			"dataType":                  table.PrimaryKey.DataType,
			"data":                      table,
		})
	if err != nil {
		return "", "", err
	}

	return output.String(), findOneMethod.String(), nil
}
