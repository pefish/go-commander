package main

import (
	"flag"
	"fmt"
	commander2 "github.com/pefish/go-commander"
	go_logger "github.com/pefish/go-logger"
	_ "net/http/pprof"
)


type TestSubCommand struct {

}

func (t TestSubCommand) DecorateFlagSet(flagSet *flag.FlagSet) error {
	flagSet.String("test", "default-flag-test", "")
	return nil
}

func (t TestSubCommand) Start(data *commander2.StartData) error {
	//fmt.Println("test", go_config.Config.MustGetString("test"))
	//fmt.Println("test-test", go_config.Config.MustGetString("test-test"))
	fmt.Println(data.Args)

	return nil
}

func (t TestSubCommand) OnExited(data *commander2.StartData) error {
	//fmt.Println("OnExited")
	return nil
}

func main() {
	//go_logger.Logger.Error(errors.WithMessage(errors.New("123"), "ywrtsdfhs"))
	commander := commander2.NewCommander("test", "v0.0.1", "小工具")
	//commander.RegisterSubcommand("test", "这是一个测试", TestSubCommand{})
	commander.RegisterSubcommand("test2", "这是一个测试", TestSubCommand{})
	commander.RegisterDefaultSubcommand("haha", TestSubCommand{})
	//commander.RegisterFnToSetCommonFlags(func(flagSet *flag.FlagSet) {
	//	flagSet.String("test-test", "", "path to config file")
	//})
	//commander.DisableSubCommand()
	err := commander.Run()
	if err != nil {
		go_logger.Logger.Error(err)
	}
}

// go run ./_example/main.go -test=cmd-test -test-test=cmd-test-test
// Output:
// test cmd-test
// test-test cmd-test-test


// go run ./_example/main.go -test-test=cmd-test-test
// Output:
// test default-flag-test
// test-test cmd-test-test

// go run ./_example/main.go -config=./_example/config.yaml -test-test=cmd-test-test
// Output:
// test config-file-test
// test-test cmd-test-test

// go run ./_example/main.go -secret-file=./_example/secret.yaml -test=cmd-test
// Output:
// test cmd-test
// test-test secret-file-test-test

// GO_SECRET=./_example/secret.yaml go run ./_example/main.go -test=cmd-test
// Output:
// test cmd-test
// test-test secret-file-test-test

// TEST=env-test go run ./_example/main.go
// Output:
// test env-test
// test-test

// TEST_TEST=env-test-test go run ./_example/main.go
// Output:
// test default-flag-test
// test-test env-test-test
