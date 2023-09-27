// Copyright 2023 Nautes Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
