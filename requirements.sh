#!/bin/bash

echo "Installing pulumi sdk"
go get github.com/pulumi/pulumi/sdk/v2/go/pulumi > /dev/null 2>&1 & 

echo "Installing pulumi-terraform sdk"
go get github.com/pulumi/pulumi-terraform/sdk/v2 > /dev/null 2>&1 & 



