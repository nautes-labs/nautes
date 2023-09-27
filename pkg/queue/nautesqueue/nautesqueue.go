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
	"fmt" //nolint:goimports
	"github.com/nautes-labs/nautes/pkg/queue"
	"k8s.io/client-go/tools/cache"
	"time" //nolint:goimports

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
)

const (
	DefaultRequeueTimes = 5
)

type clientGoWorkQueue struct {
	store        cache.ThreadSafeStore
	queue        workqueue.RateLimitingInterface
	workers      int
	requeueTimes int
	handler      func(string, []byte) error
}

func (c *clientGoWorkQueue) Send(topic string, entry []byte) {
	c.store.Add(topic, entry)
	c.queue.AddRateLimited(topic)
}

func (c *clientGoWorkQueue) AddHandler(handler func(string, []byte) error) {
	c.handler = handler
}

func NewQueue(stopCh <-chan struct{}, workers int) queue.Queuer {
	store := cache.NewThreadSafeStore(cache.Indexers{}, cache.Indices{})
	rateQueue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	controller := doNewQueue(store, rateQueue, workers, DefaultRequeueTimes)
	go controller.run(stopCh)
	return controller
}

func doNewQueue(store cache.ThreadSafeStore, rateQueue workqueue.RateLimitingInterface, workers,
	requeueTimes int) *clientGoWorkQueue {
	return &clientGoWorkQueue{
		store:        store,
		queue:        rateQueue,
		workers:      workers,
		requeueTimes: requeueTimes,
	}
}

func (c *clientGoWorkQueue) run(stopCh <-chan struct{}) {
	defer runtime.HandleCrash()

	// Let the workers stop when we are done
	defer c.queue.ShutDown()
	// klog.Infof("Starting work queue %v", c)

	for i := 0; i < c.workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	// klog.Infof("Stopping work queue %v", c)
}

func (c *clientGoWorkQueue) runWorker() {
	for c.processNextItem() {
	}
}

func (c *clientGoWorkQueue) processNextItem() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two items with the same key are never processed in
	// parallel.
	defer c.queue.Done(key)

	// Invoke the method containing the business logic
	obj, err := c.GetByKey(key.(string))
	if err != nil {
		return false
	}
	err = c.handler(key.(string), obj)
	// Handle the error if something went wrong during the execution of the business logic
	c.handleErr(err, key)
	return true
}

// GetByKey is the business logic of the controller.
func (c *clientGoWorkQueue) GetByKey(key string) ([]byte, error) {
	obj, exists := c.store.Get(key)
	if !exists {
		return nil, fmt.Errorf("the %s does not exist anymore", key)
	}
	bytes, ok := obj.([]byte)
	if !ok {
		return nil, fmt.Errorf("the %s's value does not []byte", key)
	}
	return bytes, nil
}

// handleErr checks if an error happened and makes sure we will retry later.
func (c *clientGoWorkQueue) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.queue.Forget(key)
		return
	}

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.queue.NumRequeues(key) < c.requeueTimes {
		// klog.Infof("Error syncing item %+v: %+v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.queue.AddRateLimited(key)
		return
	}

	c.queue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)

	// klog.Infof("Dropping item %q out of the queue: %+v", key, err)
}
