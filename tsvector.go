package tsvector

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/lib/pq"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// https://www.postgresql.org/docs/current/datatype-textsearch.html#DATATYPE-TSVECTOR

type TSVector struct {
	config   string
	document string
	lexemes  map[string][]int
}

func ToTSVector(args ...string) TSVector {
	switch len(args) {
	case 1:
		return TSVector{document: args[0]}
	case 2:
		return TSVector{config: args[0], document: args[1]}
	default:
		panic(fmt.Errorf("expected between 1 and 2 arguments, got: %d", len(args)))
	}
}

func (tsv TSVector) Lexemes() map[string][]int {
	return tsv.lexemes
}

// https://pkg.go.dev/database/sql#Scanner

func (tsv *TSVector) UnmarshalJSON(data []byte) error {
	if string(data) == "{}" {
		return nil
	}
	var v string
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	return tsv.Scan(v)
}

func (tsv *TSVector) Scan(v interface{}) error {

	var s string
	switch v.(type) {
	case string:
		s = fmt.Sprintf("%v", v)
	case []byte:
		b, ok := v.([]byte)
		if !ok {
			return errors.New("unexpected type from DB")
		}
		s = string(b)
	default:
		return errors.New("unexpected type from DB")

	}
	words := strings.Fields(s)
	tsv.lexemes = make(map[string][]int, len(words))
	for _, w := range words {
		splits := strings.SplitN(w, ":", 2)

		lexeme := splits[0]
		if len(lexeme) < 3 && lexeme[0] != '\'' && lexeme[len(lexeme)-1] != '\'' {
			return errors.New("expecting lexeme normalized form")
		}
		lexeme = lexeme[1 : len(lexeme)-1]

		var indices []int
		if len(splits) > 1 {
			uints := strings.Split(splits[1], ",")
			indices = make([]int, 0, len(uints))
			for _, u := range uints {
				parsed, err := strconv.Atoi(u)
				if err != nil {
					return err
				}
				indices = append(indices, parsed)
			}
		}

		tsv.lexemes[lexeme] = indices
	}

	return nil
}

// https://pkg.go.dev/database/sql/driver#Valuer

func (tsv TSVector) Value() (driver.Value, error) {
	return nil, errors.New("cannot get value")
}

// https://gorm.io/docs/data_types.html

func (tsv TSVector) GormDataType() string {
	return "tsvector"
}

func (tsv TSVector) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	if len(tsv.config) > 0 {
		return clause.Expr{
			SQL: fmt.Sprintf("to_tsvector(%s, %s)", pq.QuoteLiteral(tsv.config), pq.QuoteLiteral(tsv.document)),
			//SQL:  "to_tsvector($1, $2)",
			//Vars: []interface{}{tsv.config, tsv.document},
		}
	}
	return clause.Expr{
		SQL: fmt.Sprintf("to_tsvector(%s)", pq.QuoteLiteral(tsv.document)),
		//SQL:  "to_tsvector($2)",
		//Vars: []interface{}{tsv.document},
	}
	//return clause.Expr{}
}
