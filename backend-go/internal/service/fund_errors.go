package service

import "errors"

var (
	ErrInvalidFundCode = errors.New("invalid fund code")
	ErrFundNotFound    = errors.New("fund not found")
)
