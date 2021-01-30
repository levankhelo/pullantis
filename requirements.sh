#!/bin/bash

echo "Installing pulumi sdk"
go get github.com/pulumi/pulumi/sdk/v2/go/pulumi > /dev/null 2>&1 & 

echo "Installing pulumi-github sdk"
go get github.com/pulumi/pulumi-github/sdk/go/github > /dev/null 2>&1 & 

echo "Installing goimports"
go get golang.org/x/tools/cmd/goimports


