package error_test

import (
	"fmt"
	"testing"

	. "github.com/nautes-labs/nautes/app/api-server/util/error"
)

func TestCollectAndCombineErrors(t *testing.T) {
	errChan := make(chan interface{}, 5)

	for i := 0; i < 3; i++ {
		errChan <- fmt.Sprintf("Error %d", i+1)
	}

	err := CollectAndCombineErrors(errChan)

	expectedErrMsg := "Error 1|Error 2|Error 3"

	if err == nil {
		t.Errorf("Expected error, but got nil")
		t.Fail()
	} else if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message %q, but got %q", expectedErrMsg, err.Error())
		t.Fail()
	}
}
