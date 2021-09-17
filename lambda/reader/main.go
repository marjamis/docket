package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/marjamis/docket/lambda/reader/cmd"
)

func main() {
	lambda.Start(cmd.HandleRequest)
}
