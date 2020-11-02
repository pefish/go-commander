package main

import (
	"flag"
	"fmt"
	commander2 "github.com/pefish/go-commander"
	go_config "github.com/pefish/go-config"
	"log"
	_ "net/http/pprof"
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

// go run ./_example/main.go -test=76573 -testtest=11
// Output:
// 76573
// 11


// go run ./_example/main.go -testtest=11
// Output:
// haha
// 11

// go run ./_example/main.go -config=./_example/config.yaml -testtest=11
// Output:
// xixi
// 11

// go run ./_example/main.go -secret-file=./_example/secret.yaml -test=11
// Output:
// 11
// secretxixi

// GO_SECRET=./_example/secret.yaml go run ./_example/main.go -test=11
// Output:
// 11
// secretxixi

// TEST=22 go run ./_example/main.go
// Output:
// 22
//


