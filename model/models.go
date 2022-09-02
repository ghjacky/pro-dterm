package model

import (
	"bytes"
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

type BaseModel struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index" `
	TX        *gorm.DB       `json:"-" gorm:"-"`
}

// 分页查询
type PageQuery struct {
	Page   uint   `form:"page"`
	Size   uint   `form:"size"`
	Order  string `form:"order"`
	Search string `form:"search"` // string | ex: name:abc
	Total  int64
}

type IModel interface {
	Add() error
}

func GetColumnInTagByJsonTag(m IModel, jtag string) string {
	colPatt := regexp.MustCompile(`column:(?P<column>[^:;"]+);`)
	col := ""
	for i := 0; i < reflect.ValueOf(m).Elem().NumField(); i++ {
		tv := reflect.TypeOf(m).Elem().FieldByIndex([]int{i})
		jsontag := tv.Tag.Get("json")
		if jtag == jsontag {
			col = jsontag
			gormtag := tv.Tag.Get("gorm")
			s := colPatt.FindStringSubmatch(gormtag)
			if len(s) >= 2 {
				col = s[1]
			}
			break
		}
	}
	return col
}

func (pq *PageQuery) Query(tx *gorm.DB, vs interface{}, _vs IModel) *gorm.DB {

	if pair := strings.Split(pq.Search, ","); len(pair) > 0 {
		for _, p := range pair {
			col := GetColumnInTagByJsonTag(_vs, strings.Split(p, ":")[0])
			cts, _ := tx.Debug().Migrator().ColumnTypes(_vs)
			for _, v := range cts {
				if v.Name() == col {
					t := v.DatabaseTypeName()
					if sl := strings.Split(p, ":"); len(sl) == 2 {
						switch t {
						case "bigint", "int", "tinyint", "smallint", "mediumint", "float", "double", "boolean":
							n, _ := strconv.Atoi(sl[1])
							tx = tx.Where(fmt.Sprintf("(%s = %d)", col, n))
						default:
							tx = tx.Where(fmt.Sprintf("(%s like '%%%s%%')", col, sl[1]))
						}
					} else if len(sl) == 3 {
						switch t {
						case "bigint", "int", "tinyint", "smallint", "mediumint", "float", "double", "boolean":
							ln, _ := strconv.Atoi(sl[1])
							bn, _ := strconv.Atoi(sl[2])
							tx = tx.Where(fmt.Sprintf("(%s >= %d and %s <= %d)", col, ln, col, bn))
						default:
							tx = tx.Where(fmt.Sprintf("(%s >= '%%%s%%' and %s <= '%%%s%%')", col, sl[1], col, sl[2]))
						}
					}
					break
				}
			}
		}
	}
	tx.Model(vs).Count(&pq.Total)
	if strings.HasPrefix(pq.Order, "+") {
		tx = tx.Order(fmt.Sprintf("%s asc", strings.TrimLeft(pq.Order, "+")))
	} else if strings.HasPrefix(pq.Order, "-") {
		tx = tx.Order(fmt.Sprintf("%s desc", strings.TrimLeft(pq.Order, "-")))
	}
	return tx.Offset((int(pq.Page) - 1) * int(pq.Size)).Limit(int(pq.Size)).Find(vs)
}

// 自定义JSON类型
type JSON []byte

func (j JSON) Value() (driver.Value, error) {
	if j.IsNull() {
		return nil, nil
	}
	return string(j), nil
}
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	s, ok := value.([]byte)
	if !ok {
		return errors.New("invalid Scan Source")
	}
	*j = append((*j)[0:0], s...)
	return nil
}
func (m JSON) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	return m, nil
}
func (m *JSON) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("null point exception")
	}
	*m = append((*m)[0:0], data...)
	return nil
}
func (j JSON) IsNull() bool {
	return len(j) == 0 || string(j) == "null"
}
func (j JSON) Equals(j1 JSON) bool {
	return bytes.Equal([]byte(j), []byte(j1))
}
