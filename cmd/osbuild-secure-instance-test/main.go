package main

import (
	"fmt"
	"github.com/osbuild/osbuild-composer/internal/cloud/awscloud"
)

func main() {
	a, err := awscloud.NewDefault("us-east-1")
	if err != nil {
		panic(err)
	}
	si, err := a.RunSecureInstance("sanne-dev-secure-instance-role", "sanne-us-east-1-438669297788", "", "")
	if err != nil {
		panic(err)
	}

	fmt.Println("Instance ID: ", si.InstanceID)
}
