// Package grpclimits содержит ограничения размеров gRPC-ответов.
package grpclimits

const (
	// ResponseMessageMaxSize содержит максимальный размер исходящего gRPC-ответа в байтах.
	// Ответ списка может содержать metadata множества записей, поэтому его предел выше запроса.
	ResponseMessageMaxSize = 16 * 1024 * 1024
)
