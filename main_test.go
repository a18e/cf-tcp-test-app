package main

import "testing"

func TestHttpPathAndMethod(t *testing.T) {
	bytes := []byte("POST / HTTP/1.1\r\nHost: go-fail.cfapps.aws-cfn07.aws.cfn.sapcloud.io")
	path, method, err := getHttpPathAndMethod(bytes)
	if path != "/" {
		t.Errorf("path %s", path)
	}
	if method != "POST" {
		t.Errorf("method %s", method)
	}
	if err != nil {
		t.Errorf("err %v", err)
	}
}
