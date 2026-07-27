// Package apilimits содержит общие ограничения транспортов GophKeeper.
package apilimits

const (
	// RequestMaxSize содержит максимальный размер входящего запроса в байтах.
	// Ограничение одинаково для HTTPS и gRPC.
	RequestMaxSize = 4 * 1024 * 1024
)
