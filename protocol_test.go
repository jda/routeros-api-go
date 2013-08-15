package routeros

import (
	"os"
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
		t.Error(nil)
	}

	err = c.Connect(tv.Username, tv.Password)
	if err != nil {
		t.Error(err)
	}
}

// Test running a command (uptime)
func TestCommand(t *testing.T) {
	tv := PrepVars(t)
	c, err := New(tv.Address)
	if err != nil {
		t.Error(nil)
	}

	err = c.Connect(tv.Username, tv.Password)
	if err != nil {
		t.Error(err)
	}

	res, err := c.Call("/system/resource/getall", nil)
	if err != nil {
		t.Error(err)
	}
	uptime, err := GetPairValue(res, "uptime")
	t.Logf("Uptime: %s\n", uptime)
}

// Test querying data (getting IP addresses on ether1)
func TestQuery(t *testing.T) {
	tv := PrepVars(t)
	c, err := New(tv.Address)
	if err != nil {
		t.Error(nil)
	}

	err = c.Connect(tv.Username, tv.Password)
	if err != nil {
		t.Error(err)
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
	for _, v := range res {
		t.Log(v.Value)
	}
}
