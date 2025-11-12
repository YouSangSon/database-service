package entity

import (
	apperrors "github.com/YouSangSon/database-service/internal/pkg/errors"
)

var (
	// ErrInvalidCollection은 컬렉션명이 유효하지 않을 때 발생합니다
	ErrInvalidCollection = apperrors.ErrInvalidCollection

	// ErrInvalidData는 데이터가 유효하지 않을 때 발생합니다
	ErrInvalidData = apperrors.ErrInvalidDocument

	// ErrDocumentNotFound는 문서를 찾을 수 없을 때 발생합니다
	ErrDocumentNotFound = apperrors.ErrDocumentNotFound

	// ErrVersionConflict는 낙관적 잠금 충돌이 발생했을 때 발생합니다
	ErrVersionConflict = apperrors.ErrVersionConflict
)
