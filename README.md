# go-commander

[![view examples](https://img.shields.io/badge/learn%20by-examples-0C8EC5.svg?style=for-the-badge&logo=go)](https://github.com/pefish/go-commander)

go-commander

## Quick start

```go
package main

import (
	"flag"
	"fmt"
	commander2 "github.com/pefish/go-commander"
	go_config "github.com/pefish/go-config"
	"log"
)


type TestSubCommand struct {

}

func (t TestSubCommand) DecorateFlagSet(flagSet *flag.FlagSet) error {
	flagSet.String("test", "haha", "")
	return nil
}

func (t TestSubCommand) Start() error {
	fmt.Println(go_config.Config.MustGetString("test"))
	fmt.Println(go_config.Config.MustGetString("testtest"))
	return nil
}

func main() {
	commander := commander2.NewCommander("test", "v0.0.1", "")
	commander.RegisterSubcommand("test", TestSubCommand{})
	commander.RegisterDefaultSubcommand(TestSubCommand{})
	commander.RegisterFnToSetCommonFlags(func(flagSet *flag.FlagSet) {
		flagSet.String("testtest", "", "path to config file")
	})
	err := commander.Run()
	if err != nil {
		log.Fatal(err)
	}
}

```

```shell script
go run ./_example/main.go -test=76573 -testtest=11
```

## Document

[doc](https://godoc.org/github.com/pefish/go-commander)

## Security Vulnerabilities

If you discover a security vulnerability, please send an e-mail to [pefish@qq.com](mailto:pefish@qq.com). All security vulnerabilities will be promptly addressed.

## License

This project is licensed under the [Apache License](LICENSE).
