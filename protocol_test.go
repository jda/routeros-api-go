package routeros

import (
	"os"
	"strconv"
	"testing"
)

type TestVars struct {
	Username string
	Password string
	Address  string
}

// Make sure we have the env vars to run, handle bailing if we don't
func PrepVars(t *testing.T) TestVars {
	var tv TestVars

	addr := os.Getenv("ROS_TEST_TARGET")
	if addr == "" {
		t.Skip("Can't run test because ROS_TEST_TARGET undefined")
	} else {
		tv.Address = addr
	}

	username := os.Getenv("ROS_TEST_USER")
	if username == "" {
		tv.Username = "admin"
		t.Logf("ROS_TEST_USER not defined. Assuming %s\n", tv.Username)
	} else {
		tv.Username = username
	}

	password := os.Getenv("ROS_TEST_PASSWORD")
	if password == "" {
		tv.Password = "admin"
		t.Logf("ROS_TEST_PASSWORD not defined. Assuming %s\n", tv.Password)
	} else {
		tv.Password = password
	}

	return tv
}

// Test logging in and out
func TestLogin(t *testing.T) {
	tv := PrepVars(t)
	c, err := New(tv.Address)
	if err != nil {
		t.Fatal(err)
	}

	err = c.Connect(tv.Username, tv.Password)
	if err != nil {
		t.Fatal(err)
	}
}

// Test running a command (uptime)
func TestCommand(t *testing.T) {
	tv := PrepVars(t)
	c, err := New(tv.Address)
	if err != nil {
		t.Fatal(err)
	}

	err = c.Connect(tv.Username, tv.Password)
	if err != nil {
		t.Fatal(err)
	}

	res, err := c.Call("/system/resource/getall", nil)
	if err != nil {
		t.Error(err)
	}

	uptime := res.SubPairs[0]["uptime"]
	t.Logf("Uptime: %s\n", uptime)
}

// Test querying data (getting IP addresses on ether1)
func TestQuery(t *testing.T) {
	tv := PrepVars(t)
	c, err := New(tv.Address)
	if err != nil {
		t.Fatal(err)
	}

	err = c.Connect(tv.Username, tv.Password)
	if err != nil {
		t.Fatal(err)
	}

	getEther1Addrs := NewPair("interface", "ether1")
	getEther1Addrs.Op = "="
	var q Query
	q.Pairs = append(q.Pairs, *getEther1Addrs)
	q.Proplist = []string{"address"}

	res, err := c.Query("/ip/address/print", q)
	if err != nil {
		t.Error(err)
	}

	t.Log("IP addresses on ether1:")
	for _, v := range res.SubPairs {
		for _, sv := range v {
			t.Log(sv)
		}
	}
}

// Test adding some bridges (test of Call)
func TestCallAddBridges(t *testing.T) {
	tv := PrepVars(t)
	c, err := New(tv.Address)
	if err != nil {
		t.Fatal(err)
	}

	err = c.Connect(tv.Username, tv.Password)
	if err != nil {
		t.Fatal(err)
	}

	for i := 1; i <= 10; i++ {
		var pairs []Pair
		bName := "test-bridge" + strconv.Itoa(i)
		pairs = append(pairs, Pair{Key: "name", Value: bName})
		pairs = append(pairs, Pair{Key: "comment", Value: "test bridge number " + strconv.Itoa(i)})
		pairs = append(pairs, Pair{Key: "arp", Value: "disabled"})
		res, err := c.Call("/interface/bridge/add", pairs)
		if err != nil {
			t.Errorf("Error adding bridge: %s\n", err)
		}
		t.Logf("reply from adding bridge: %+v\n", res)
	}
}

// Test getting list of interfaces (test Query)
func TestQueryMultiple(t *testing.T) {
	tv := PrepVars(t)
	c, err := New(tv.Address)
	if err != nil {
		t.Fatal(err)
	}

	err = c.Connect(tv.Username, tv.Password)
	if err != nil {
		t.Fatal(err)
	}

	var q Query
	q.Pairs = append(q.Pairs, Pair{Key: "type", Value: "bridge", Op: "="})

	res, err := c.Query("/interface/print", q)
	if err != nil {
		t.Error(err)
	}
	if len(res.SubPairs) <= 1 {
		t.Error("Did not get multiple SubPairs from bridge interface query")
	}
}

// Test query with proplist
func TestQueryWithProplist(t *testing.T) {
	tv := PrepVars(t)
	c, err := New(tv.Address)
	if err != nil {
		t.Fatal(err)
	}

	err = c.Connect(tv.Username, tv.Password)
	if err != nil {
		t.Fatal(err)
	}

	var q Query
	q.Proplist = append(q.Proplist, "name")
	q.Proplist = append(q.Proplist, "comment")
	q.Proplist = append(q.Proplist, ".id")
	q.Pairs = append(q.Pairs, Pair{Key: "type", Value: "bridge", Op: "="})
	res, err := c.Query("/interface/print", q)
	if err != nil {
		t.Fatal(err)
	}

	for _, b := range res.SubPairs {
		t.Logf("Found bridge %s (%s)\n", b["name"], b["comment"])

	}
}

// Test query with proplist
func TestCallRemoveBridges(t *testing.T) {
	tv := PrepVars(t)
	c, err := New(tv.Address)
	if err != nil {
		t.Fatal(err)
	}

	err = c.Connect(tv.Username, tv.Password)
	if err != nil {
		t.Fatal(err)
	}

	var q Query
	q.Proplist = append(q.Proplist, ".id")
	q.Pairs = append(q.Pairs, Pair{Key: "type", Value: "bridge", Op: "="})
	res, err := c.Query("/interface/print", q)
	if err != nil {
		t.Fatal(err)
	}

	for _, v := range res.SubPairs {
		var pairs []Pair
		pairs = append(pairs, Pair{Key: ".id", Value: v[".id"]})
		_, err = c.Call("/interface/bridge/remove", pairs)
		if err != nil {
			t.Errorf("error removing bridge: %s\n", err)
		}
	}
}

// Test call that should trigger error response from router
func TestCallCausesError(t *testing.T) {
	tv := PrepVars(t)
	c, err := New(tv.Address)
	if err != nil {
		t.Fatal(err)
	}

	err = c.Connect(tv.Username, tv.Password)
	if err != nil {
		t.Fatal(err)
	}

	var pairs []Pair
	pairs = append(pairs, Pair{Key: "address", Value: "192.168.99.1/32"})
	pairs = append(pairs, Pair{Key: "comment", Value: "this address should never be added"})
	pairs = append(pairs, Pair{Key: "interface", Value: "badbridge99"})
	_, err = c.Call("/ip/address/add", pairs)
	if err != nil {
		t.Logf("Error adding address to nonexistent bridge: %s\n", err)
	} else {
		t.Error("did not get error when adding address to nonexistent bridge")
	}
}

// Test query that should trigger error response from router
func TestQueryCausesError(t *testing.T) {
	tv := PrepVars(t)
	c, err := New(tv.Address)
	if err != nil {
		t.Fatal(err)
	}

	err = c.Connect(tv.Username, tv.Password)
	if err != nil {
		t.Fatal(err)
	}

	var q Query
	q.Proplist = append(q.Proplist, ".id")
	_, err = c.Query("/ip/address/sneeze", q)
	if err != nil {
		t.Logf("Error querying with nonexistent command: %s\n", err)
	} else {
		t.Error("did not get error when querying nonexistent command")
	}
}
