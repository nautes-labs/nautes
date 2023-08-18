package error

import (
	"fmt"
	"strings"
)

func CollectAndCombineErrors(errChan chan interface{}) error {
	close(errChan)

	errMsgs := make([]string, 0)

	for err := range errChan {
		switch err := err.(type) {
		case string:
			errMsgs = append(errMsgs, err)
		case error:
			errMsgs = append(errMsgs, err.Error())
		default:
			errMsgs = append(errMsgs, fmt.Sprintf("Unknown error type: %v", err))
		}
	}

	if len(errMsgs) > 0 {
		return fmt.Errorf(strings.Join(errMsgs, "|"))
	}

	return nil
}
