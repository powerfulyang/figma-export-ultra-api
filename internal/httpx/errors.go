package httpx

import "fiber-ent-apollo-pg/internal/httpx/kit"

// ErrorHandler exposes the kit error handler to keep main.go API stable.
var ErrorHandler = kit.ErrorHandler
