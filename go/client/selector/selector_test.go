package selector

import (
	"testing"
)

func TestSelectAll(t *testing.T) {
	testServers := []string{"a", "b", "c"}
	sa := new(All)
	if err := sa.Set(testServers...); err != nil {
		t.Fatal(err)
	}
	servers, err := sa.Get("test")
	if err != nil {
		t.Fatal(err)
	}
	for _, tserver := range testServers {
		var seen bool
		for _, server := range servers {
			if server == tserver {
				seen = true
				break
			}
		}
		if !seen {
			t.Fatal("did not find", tserver)
		}
	}
}

func TestSelectShard(t *testing.T) {
	testServers := []string{"a", "b", "c"}
	ss := new(Shard)
	if err := ss.Set(testServers...); err != nil {
		t.Fatal(err)
	}
	servers, err := ss.Get("test")
	if err != nil {
		t.Fatal(err)
	}
	var seen bool
	for _, tserver := range testServers {
		for _, server := range servers {
			if server == tserver {
				seen = true
				break
			}
		}
	}
	if !seen {
		t.Fatal("did not find any test servers")
	}
}
