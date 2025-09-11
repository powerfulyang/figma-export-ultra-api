package httpx

import "fiber-ent-apollo-pg/internal/httpx/kit"

// Wrappers to keep main.go API stable
var ErrorHandler = kit.ErrorHandler
