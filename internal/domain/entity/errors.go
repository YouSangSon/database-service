package entity

import "errors"

var (
	// ErrInvalidCollection은 컬렉션명이 유효하지 않을 때 발생합니다
	ErrInvalidCollection = errors.New("invalid collection name")

	// ErrInvalidData는 데이터가 유효하지 않을 때 발생합니다
	ErrInvalidData = errors.New("invalid data")

	// ErrDocumentNotFound는 문서를 찾을 수 없을 때 발생합니다
	ErrDocumentNotFound = errors.New("document not found")

	// ErrVersionConflict는 낙관적 잠금 충돌이 발생했을 때 발생합니다
	ErrVersionConflict = errors.New("version conflict")
)
