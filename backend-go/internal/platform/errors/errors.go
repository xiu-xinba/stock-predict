package errors

type Code int

const (
	CodeSuccess         Code = 0
	CodeBadRequest      Code = -1
	CodeFeatureDisabled Code = -2
)
