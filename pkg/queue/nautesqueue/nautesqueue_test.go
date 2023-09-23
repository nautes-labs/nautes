package nautesqueue

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

type body struct {
	Token   string
	Cluster string
}

func TestNewQueue(t *testing.T) {

	// ######## how use start
	stop := make(chan struct{})
	defer close(stop)

	controller := NewQueue(stop, 1)
	controller.AddHandler(func(topic string, obj []byte) error {
		// you are logic
		b := &body{}
		err := json.Unmarshal(obj, b)
		if err != nil {
			fmt.Println(err)
			return err
		}
		fmt.Printf("workqueue: topic: %s, body: %+v\n", topic, b)
		return nil
	})
	// ########## how use end

	// test send
	b := body{
		Token:   "gitlab access token",
		Cluster: "important cluster information",
	}

	bytes, err := json.Marshal(b)
	if err != nil {
		fmt.Println(err)
		return
	}

	// test receive 3
	i := 0
	for {
		if i > 3 {
			break
		}
		i++
		controller.Send("topic-101", bytes)
		time.Sleep(500 * time.Millisecond)
	}
}
