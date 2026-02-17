package response

import (
	"encoding/json"
	"time"

	"notification-srv/pkg/errors"
)

type Resp struct {
	ErrorCode int    `json:"error_code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	Errors    any    `json:"errors,omitempty"`
}

type ErrorMapping map[error]*errors.HTTPError

type Date time.Time

func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(d).Local().Format(DateFormat))
}

type DateTime time.Time

func (d DateTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(d).Local().Format(DateTimeFormat))
}
