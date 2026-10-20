package events

import (
	"sync"
	"testing"
	"time"
)

func TestEventBusPubSub(t *testing.T) {
	bus := GetBus()

	var wg sync.WaitGroup
	wg.Add(1)

	var receivedTopic string
	var receivedPayload any

	bus.Subscribe("test/topic", func(ev Event) {
		receivedTopic = ev.Topic
		receivedPayload = ev.Payload
		wg.Done()
	})

	bus.Publish("test/topic", "hello event")

	// Wait with timeout
	c := make(chan struct{})
	go func() {
		wg.Wait()
		close(c)
	}()

	select {
	case <-c:
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for event")
	}

	if receivedTopic != "test/topic" {
		t.Errorf("expected topic test/topic, got %s", receivedTopic)
	}
	if receivedPayload != "hello event" {
		t.Errorf("expected payload 'hello event', got %v", receivedPayload)
	}
}

func TestEventBusWildcards(t *testing.T) {
	bus := GetBus()

	var wg sync.WaitGroup
	wg.Add(2)

	var mutex sync.Mutex
	receivedTopics := make(map[string]bool)

	// Subscribe to docker/* wildcard
	bus.Subscribe("docker/*", func(ev Event) {
		mutex.Lock()
		receivedTopics[ev.Topic] = true
		mutex.Unlock()
		wg.Done()
	})

	bus.Publish("docker/container/start", "container-1")
	bus.Publish("docker/container/stop", "container-2")
	// Should not match:
	bus.Publish("system/cpu", "high")

	c := make(chan struct{})
	go func() {
		wg.Wait()
		close(c)
	}()

	select {
	case <-c:
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for wildcard events")
	}

	mutex.Lock()
	defer mutex.Unlock()
	if !receivedTopics["docker/container/start"] {
		t.Error("failed to receive docker/container/start")
	}
	if !receivedTopics["docker/container/stop"] {
		t.Error("failed to receive docker/container/stop")
	}
	if receivedTopics["system/cpu"] {
		t.Error("received system/cpu which should not match docker/* wildcard")
	}
}

func TestEventBusUnsubscribe(t *testing.T) {
	bus := GetBus()

	var count int
	var mutex sync.Mutex

	id := bus.Subscribe("test/unsub", func(ev Event) {
		mutex.Lock()
		count++
		mutex.Unlock()
	})

	bus.Publish("test/unsub", "first")
	time.Sleep(50 * time.Millisecond)

	bus.Unsubscribe(id)

	bus.Publish("test/unsub", "second")
	time.Sleep(50 * time.Millisecond)

	mutex.Lock()
	defer mutex.Unlock()
	if count != 1 {
		t.Errorf("expected count to be 1, got %d", count)
	}
}
