package clock

import "time"

type UTC struct{}
func (UTC) Now() time.Time { return time.Now().UTC() }
