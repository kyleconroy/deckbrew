package main

import (
	"fmt"
	"strconv"
	"strings"
)

// users := Select("*").From("users")
// active := users.Where(Eq{"deleted_at": nil})

const (
	PARAM = "$%d"
)

type Condition interface {
	ToSql() (string, []interface{}, error)
}

type Expression struct {
	Fields    []string
	Tables    []string
	Action    string
	By        string
	limit     int
	offset    int
	Condition Condition
}

type comparison struct {
	column   string
	operator string
	val      interface{}
}

func (c comparison) ToSql() (string, []interface{}, error) {
	sql := fmt.Sprintf("%s %s %s", c.column, c.operator, PARAM)
	return sql, []interface{}{c.val}, nil
}

type multicompare struct {
	operator    string
	comparisons []Condition
}

func (m multicompare) ToSql() (string, []interface{}, error) {
	args := []interface{}{}
	query := ""

	for _, comp := range m.comparisons {
		subquery, subargs, err := comp.ToSql()

		if err != nil {
			return query, args, err
		}

		for _, a := range subargs {
			args = append(args, a)
		}

		if query == "" {
			query = subquery
		} else {
			query = query + " " + m.operator + " " + subquery
		}
	}

	return "(" + query + ")", args, nil
}

func And(conds ...Condition) Condition {
	return multicompare{operator: "AND", comparisons: conds}
}

func Or(conds ...Condition) Condition {
	return multicompare{operator: "OR", comparisons: conds}
}

func Eq(column string, val interface{}) Condition {
	return comparison{column: column, operator: "=", val: val}
}

func Gt(column string, val interface{}) Condition {
	return comparison{column: column, operator: ">", val: val}
}

func Lt(column string, val interface{}) Condition {
	return comparison{column: column, operator: "<", val: val}
}

func Gte(column string, val interface{}) Condition {
	return comparison{column: column, operator: ">=", val: val}
}

func Lte(column string, val interface{}) Condition {
	return comparison{column: column, operator: "<=", val: val}
}

func Overlap(column string, val interface{}) Condition {
	return comparison{column: column, operator: "&&", val: val}
}

func Contains(column string, val interface{}) Condition {
	return comparison{column: column, operator: "@>", val: val}
}

func ILike(column string, val interface{}) Condition {
	return comparison{column: column, operator: "ILIKE", val: val}
}

func (e Expression) From(table ...string) Expression {
	e.Tables = table
	return e
}

func (e Expression) Where(cond Condition) Expression {
	e.Condition = cond
	return e
}

func (e Expression) Offset(count int) Expression {
	e.offset = count
	return e
}

func (e Expression) Limit(count int) Expression {
	e.limit = count
	return e
}

func (e Expression) OrderBy(value string, asc bool) Expression {
	e.By = value
	return e
}

func (e Expression) ToSql() (string, []interface{}, error) {
	params := []interface{}{}
	if len(e.Fields) == 0 {
		return "", params, fmt.Errorf("No fields in SQL expression")
	}

	if len(e.Tables) == 0 {
		return "", params, fmt.Errorf("No tables to query in SQL expression")
	}

	sql := e.Action + " " + strings.Join(e.Fields, ", ")
	sql = sql + " FROM " + strings.Join(e.Tables, ", ")

	if e.Condition != nil {
		csql, args, err := e.Condition.ToSql()

		if err != nil {
			return sql, params, nil
		}

		if csql != "()" {

			sql = sql + " WHERE " + csql

			for _, arg := range args {
				params = append(params, arg)
			}
		}
	}

	if e.By != "" {
		sql = sql + " ORDER BY " + e.By + " ASC"
	}

	if e.limit != 0 {
		sql = sql + " LIMIT " + strconv.Itoa(e.limit)
	}

	if e.offset != 0 {
		sql = sql + " OFFSET " + strconv.Itoa(e.offset)
	}

	counts := []interface{}{}

	for i, _ := range params {
		counts = append(counts, i+1)
	}
	sql = fmt.Sprintf(sql, counts...)

	return sql, params, nil
}

// This doesn't match Select at all
func Insert(columns []string, table string) string {
	values := []string{}

	for i := range columns {
		values = append(values, "$"+strconv.Itoa(i+1))
	}

	c := strings.Join(columns, ",")
	v := strings.Join(values, ",")

	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, c, v)
}

func Update(id, columns []string, table string) string {
	values := []string{}

	for i := range columns {
		values = append(values, "$"+strconv.Itoa(i+1))
	}

	c := strings.Join(columns, ",")
	v := strings.Join(values, ",")

	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, c, v)
}

func Select(fields ...string) Expression {
	return Expression{Action: "SELECT", Fields: fields}
}
