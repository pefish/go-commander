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
