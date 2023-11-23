package tsvector_test

import (
	"database/sql"
	"github.com/dakaheni/go-tsvector"
	"github.com/go-test/deep"
	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"testing"
	"time"
)

var sqlDB *sql.DB
var gormDB *gorm.DB

func init() {
	dsn := "host=localhost user=tsvector password=tsvector dbname=tsvector sslmode=disable"

	if db, err := sql.Open("postgres", dsn); err != nil {
		panic(err)
	} else {
		sqlDB = db
	}

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Info, // Log level
			IgnoreRecordNotFoundError: false,       // Ignore ErrRecordNotFound error for logger
			Colorful:                  true,        // Enable color
		},
	)

	if db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: newLogger,
	}); err != nil {
		panic(err)
	} else {
		gormDB = db
	}

	if err := gormDB.Migrator().DropTable(
		&tsvectorTestModel{},
		&tsvectorTestGormModel{},
	); err != nil {
		panic(err)
	}

	if err := gormDB.AutoMigrate(
		&tsvectorTestModel{},
		&tsvectorTestGormModel{},
	); err != nil {
		panic(err)
	}
}

type BaseModel struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	//UpdatedAt time.Time
	//DeletedAt gorm.DeletedAt `gorm:"index"`
}

type tsvectorTestModel struct {
	ID      uint              `gorm:"primaryKey"`
	Text    string            `gorm:"not null"`
	TextTSV tsvector.TSVector `gorm:"not null"`
}

type tsvectorTestGormModel struct {
	gorm.Model
	Text    string            `gorm:"not null"`
	TextTSV tsvector.TSVector `gorm:"not null"`
}

func (m *tsvectorTestGormModel) BeforeSave(*gorm.DB) (err error) {

	m.TextTSV = tsvector.ToTSVector("english", m.Text)

	return nil
}

func TestTSVectorSQLCast(t *testing.T) {
	var tsvector tsvector.TSVector
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

func TestTSVectorSQLScan(t *testing.T) {
	var tsvector tsvector.TSVector
	err := sqlDB.
		QueryRow("SELECT TO_TSVECTOR($1)", "I am a test: the quick brown fox jumps over the lazy fox!").
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

func TestTSVectorGORMCreateFindWithoutBaseModel(t *testing.T) {
	text := "I am a test: the quick brown fox jumps over the lazy fox!"

	in := tsvectorTestModel{
		Text:    text,
		TextTSV: tsvector.ToTSVector(text),
	}
	res := gormDB.Create(&in)
	if res.Error != nil {
		t.Error(res.Error)
	}

	var out tsvectorTestModel
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

func TestTSVectorGORMCreateFindWithBaseModel(t *testing.T) {
	text := "I am a test: the quick brown fox jumps over the lazy fox!"

	in := tsvectorTestGormModel{
		Text: text,
	}
	res := gormDB.Create(&in)
	if res.Error != nil {
		t.Error(res.Error)
	}

	var out tsvectorTestGormModel
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
