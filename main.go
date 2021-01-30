package main

// import libraries
import (
	"fmt" // std
	"os"
	"github.com/pulumi/pulumi/sdk/v2/go/pulumi"  // pulumi
	"github.com/pulumi/pulumi-terraform/sdk/v2/go/" // terraform
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		fmt.Println("NULL");
		return nil;
	})
}
