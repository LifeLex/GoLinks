package links

// InvalidQueryError is returned when a request can't be satisfied because of bad
// input (unknown keyword, malformed URL, validation failure). The HTTP layer maps
// it to 400; everything else is treated as 500.
type InvalidQueryError struct {
	Message string
}

func (e InvalidQueryError) Error() string {
	return e.Message
}
