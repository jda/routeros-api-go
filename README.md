routeros-api-go
===============

Go library to manage Mikrotik routers using the Mikrotik RouterOS API

[![GoDoc](https://godoc.org/github.com/jda/routeros-api-go?status.png)](http://godoc.org/github.com/jda/routeros-api-go)

# Usage
```go
import (
    "github.com/jda/routeros-api-go"
    "fmt"
)
c, err := routeros.New("10.0.0.1:8728")
if err != nil {
    fmt.Errorf("Error parsing address: %s\n", err)
}

err = c.Connect("username", "password")
if err != nil {
    fmt.Errorf("Error connecting to device: %s\n", err)
}

res, err := c.Call("/system/resource/getall", nil)
if err != nil {
    fmt.Errorf("Error getting system resources: %s\n", err)
}

uptime := res.SubPairs[0]["uptime"]
fmt.Printf("Uptime: %s\n", uptime)
```

# Running Tests
You need a device or VM running Mikrotik RouterOS to run tests. Mikrotik provides VM images of RouterOS under the [Cloud Hosted Router](http://www.mikrotik.com/download#chr)(CHR) brand. The free edition of CHR is limited to 1Mbps per interface which is more than sufficient for API testing.

I run the VMDK under VMware Fusion with host-only networking. The CHR images are running DHCP client by default so there's no network setup required, just log in to the image and "/ip address print" to discover which address to use. Change the password for the admin user from blank to admin (or whatever you chose): "/user set 0 password=admin"

## Test setup
export ROS_TEST_TARGET=VM_IP:API_PORT
export ROS_TEST_USER=admin
export ROS_TEST_PASSWORD=admin

## To Do
* Write better docstrings
* Make README
* Add support for command/response tags
* Add checking for error codes
