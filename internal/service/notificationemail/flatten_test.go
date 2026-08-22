package notificationemail

import (
	"testing"
)

func TestFlatten_NilAPI(t *testing.T) {
	t.Parallel()
	var m model
	if err := flatten(nil, &m); err == nil {
		t.Fatal("flatten(nil) want error")
	}
}
