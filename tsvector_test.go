package pgtypes_test

import (
	"database/sql"
	"testing"

	"github.com/aymericbeaumet/go-pgtypes"
	"github.com/go-test/deep"
	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type textsearchTestModel struct {
	ID      uint             `gorm:"primaryKey"`
	Text    string           `gorm:"not null"`
	TextTSV pgtypes.TSVector `gorm:"not null"`
}

var sqlDB *sql.DB
var gormDB *gorm.DB

func init() {
	dsn := "host=localhost user=pgtypes password=pgtypes dbname=pgtypes sslmode=disable"

	if db, err := sql.Open("postgres", dsn); err != nil {
		panic(err)
	} else {
		sqlDB = db
	}

	if db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{}); err != nil {
		panic(err)
	} else {
		gormDB = db
	}

	if err := gormDB.AutoMigrate(
		&textsearchTestModel{},
	); err != nil {
		panic(err)
	}
}

func TestSQLTSVectorCast(t *testing.T) {
	var tsvector pgtypes.TSVector
	err := sqlDB.
		QueryRow("SELECT $1::tsvector", "The quick brown fox jumps over the lazy dog").
		Scan(&tsvector)
	if err != nil {
		t.Error(err)
	}

	expected := map[string][]int{
		"The":   nil,
		"brown": nil,
		"dog":   nil,
		"fox":   nil,
		"jumps": nil,
		"lazy":  nil,
		"over":  nil,
		"quick": nil,
		"the":   nil,
	}
	if diff := deep.Equal(tsvector.Lexemes(), expected); diff != nil {
		t.Error(diff)
	}
}

func TestSQLToTSVector(t *testing.T) {
	var tsvector pgtypes.TSVector
	err := sqlDB.
		QueryRow("SELECT to_tsvector($1)", "I am a test: the quick brown fox jumps over the lazy fox!").
		Scan(&tsvector)
	if err != nil {
		t.Error(err)
	}

	expected := map[string][]int{
		"brown": {7},
		"fox":   {8, 13},
		"jump":  {9},
		"lazi":  {12},
		"quick": {6},
		"test":  {4},
	}
	if diff := deep.Equal(tsvector.Lexemes(), expected); diff != nil {
		t.Error(diff)
	}
}

func TestGORMTSVector(t *testing.T) {
	text := "I am a test: the quick brown fox jumps over the lazy fox!"

	in := textsearchTestModel{
		Text:    text,
		TextTSV: pgtypes.ToTSVector(text),
	}
	res := gormDB.Create(&in)
	if res.Error != nil {
		t.Error(res.Error)
	}

	var out textsearchTestModel
	if res := gormDB.First(&out, in.ID); res.Error != nil {
		t.Error(res.Error)
	}

	expected := map[string][]int{
		"brown": {7},
		"fox":   {8, 13},
		"jump":  {9},
		"lazi":  {12},
		"quick": {6},
		"test":  {4},
	}
	if diff := deep.Equal(out.TextTSV.Lexemes(), expected); diff != nil {
		t.Error(diff)
	}
}
