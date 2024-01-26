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

func (t TestSubCommand) Init(command *commander2.Commander) error {
	return nil
}

func (t TestSubCommand) Start(command *commander2.Commander) error {
	go_logger.Logger.InfoFRaw("command: %s, test param: %s, args: %#v", command.Name, go_config.ConfigManagerInstance.MustString("test"), command.Args)
	return nil
}

func (t TestSubCommand) OnExited(data *commander2.Commander) error {
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

// go run ./_example test2 -- 1.txt 2.txt
// go run ./_example -- 1.txt
// go run ./_example --test="sgdfgs" -- 1.txt
