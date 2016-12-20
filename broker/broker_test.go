package broker

import (
	"fmt"
	"sync"
	"testing"
)

func TestBroker(t *testing.T) {
	b := New()

	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		topic := fmt.Sprintf("test%d", i)
		payload := []byte(topic)

		ch, err := b.Subscribe(topic)
		if err != nil {
			t.Fatal(err)
		}

		wg.Add(1)

		go func() {
			e := <-ch
			if string(e) != string(payload) {
				t.Fatalf("%s expected %s got %s", topic, string(payload), string(e))
			}
			if err := b.Unsubscribe(topic, ch); err != nil {
				t.Fatal(err)
			}
			wg.Done()
		}()

		if err := b.Publish(topic, payload); err != nil {
			t.Fatal(err)
		}
	}

	wg.Wait()
}
