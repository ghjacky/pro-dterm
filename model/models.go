package model

import (
	"bytes"
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

type BaseModel struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deletedAt" gorm:"index" `
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

func (pq *PageQuery) Query(tx *gorm.DB, vs interface{}) *gorm.DB {

	if pair := strings.Split(pq.Search, ","); len(pair) > 0 {
		for _, p := range pair {
			if sl := strings.Split(p, ":"); len(sl) == 2 {
				switch _vs := reflect.ValueOf(vs).Elem().Interface().(type) {
				case []interface{}:
					switch reflect.ValueOf(_vs[0]).Elem().FieldByName(sl[0]).Interface().(type) {
					case string:
						tx = tx.Where(fmt.Sprintf("%s like '%%%s%%'", sl[0], sl[1]))
					case int, int64, int8, int32, float32, float64:
						n, _ := strconv.Atoi(sl[1])
						tx = tx.Where(fmt.Sprintf("%s = %d", sl[0], n))
					}
				default:
					tx = tx.Where(fmt.Sprintf("%s like '%%%s%%'", sl[0], sl[1]))
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
