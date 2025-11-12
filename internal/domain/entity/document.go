package entity

import (
	"time"
)

// Document는 도메인 엔티티입니다
type Document struct {
	id         string
	collection string
	data       map[string]interface{}
	version    int
	createdAt  time.Time
	updatedAt  time.Time
}

// NewDocument는 새로운 Document 엔티티를 생성합니다
func NewDocument(collection string, data map[string]interface{}) (*Document, error) {
	if collection == "" {
		return nil, ErrInvalidCollection
	}
	if data == nil {
		return nil, ErrInvalidData
	}

	now := time.Now()
	return &Document{
		collection: collection,
		data:       data,
		version:    1,
		createdAt:  now,
		updatedAt:  now,
	}, nil
}

// ReconstructDocument는 기존 데이터로부터 Document를 재구성합니다 (persistence layer용)
func ReconstructDocument(id, collection string, data map[string]interface{}, version int, createdAt, updatedAt time.Time) *Document {
	return &Document{
		id:         id,
		collection: collection,
		data:       data,
		version:    version,
		createdAt:  createdAt,
		updatedAt:  updatedAt,
	}
}

// ID는 문서 ID를 반환합니다
func (d *Document) ID() string {
	return d.id
}

// Collection은 컬렉션명을 반환합니다
func (d *Document) Collection() string {
	return d.collection
}

// Data는 문서 데이터를 반환합니다
func (d *Document) Data() map[string]interface{} {
	// 방어적 복사
	result := make(map[string]interface{})
	for k, v := range d.data {
		result[k] = v
	}
	return result
}

// Version은 문서 버전을 반환합니다 (낙관적 잠금용)
func (d *Document) Version() int {
	return d.version
}

// CreatedAt은 생성 시간을 반환합니다
func (d *Document) CreatedAt() time.Time {
	return d.createdAt
}

// UpdatedAt은 수정 시간을 반환합니다
func (d *Document) UpdatedAt() time.Time {
	return d.updatedAt
}

// Update는 문서 데이터를 업데이트합니다
func (d *Document) Update(data map[string]interface{}) error {
	if data == nil {
		return ErrInvalidData
	}

	d.data = data
	d.updatedAt = time.Now()
	d.version++

	return nil
}

// SetID는 ID를 설정합니다 (persistence layer에서만 사용)
func (d *Document) SetID(id string) {
	d.id = id
}
