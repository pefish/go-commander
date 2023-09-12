package main

import (
	"flag"
	commander2 "github.com/pefish/go-commander"
	go_config "github.com/pefish/go-config"
	go_logger "github.com/pefish/go-logger"
	_ "net/http/pprof"
)

type TestSubCommand struct {
}

func (t TestSubCommand) DecorateFlagSet(flagSet *flag.FlagSet) error {
	flagSet.String("test", "default-flag-test", "")
	return nil
}

func (t TestSubCommand) Init(data *commander2.StartData) error {
	return nil
}

func (t TestSubCommand) Start(data *commander2.StartData) error {
	go_logger.Logger.InfoFRaw("test: %s, args: %#v", go_config.ConfigManagerInstance.MustGetString("test"), data.Args)
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
	commander.RegisterSubcommand("test2", &commander2.SubcommandInfo{
		Desc: "这是一个测试",
		Args: []string{
			"arg1",
			"arg2",
		},
		Subcommand: TestSubCommand{},
	})
	commander.RegisterDefaultSubcommand(&commander2.SubcommandInfo{
		Desc: "haha",
		Args: []string{
			"file",
		},
		Subcommand: TestSubCommand{},
	})
	//commander.RegisterFnToSetCommonFlags(func(flagSet *flag.FlagSet) {
	//	flagSet.String("test-test", "", "path to config file")
	//})
	//commander.DisableSubCommand()
	err := commander.Run()
	if err != nil {
		go_logger.Logger.ErrorFRaw("%s", err.Error())
	}
}
