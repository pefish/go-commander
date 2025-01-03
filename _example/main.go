package main

import (
	"log"
	_ "net/http/pprof"

	commander2 "github.com/pefish/go-commander"
	go_config "github.com/pefish/go-config"
)

type Config struct {
	Test    string `json:"test" default:"default-flag-test" usage:"test flag set"`
	FuckInt int    `json:"fuck-int" default:"888" usage:"test fuck int"`
	Abc     int    `json:"abc" default:"888" usage:"test fuck int"`
	commander2.BasicConfig
}

var GlobalConfig Config

var Data struct {
	Data1 string `json:"data1"`
}

type TestSubCommand struct {
}

func (t TestSubCommand) Config() interface{} {
	return &GlobalConfig
}

func (t TestSubCommand) Data() interface{} {
	return &Data
}

func (t TestSubCommand) Init(command *commander2.Commander) error {
	return nil
}

func (t TestSubCommand) Start(command *commander2.Commander) error {
	command.Logger.InfoFRaw(
		"command: %s, test param: %s, args: %#v",
		command.Name,
		go_config.ConfigManagerInstance.MustString("test"),
		command.Args,
	)
	command.Logger.InfoFRaw("GlobalConfig: %#v", GlobalConfig)
	command.Logger.InfoFRaw("Data: %#v", Data)
	Data.Data1 = "data2"
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
		Desc: "这是一个 test2 子命令",
		Args: []string{
			"arg1",
			"arg2",
		},
		Subcommand: TestSubCommand{},
	})
	commander.RegisterDefaultSubcommand(&commander2.SubcommandInfo{
		Desc: "这是默认命令",
		Args: []string{
			"file",
		},
		Subcommand: TestSubCommand{},
	})

	err := commander.Run()
	if err != nil {
		log.Fatal(err)
	}
}

// go run ./_example test2 --version
// go run ./_example --help
// go run ./_example test2 --help
// go run ./_example test2 -- 1.txt 2.txt
// go run ./_example -- 1.txt
// go run ./_example --test="sgdfgs" -- 1.txt
// TEST=env-test go run ./_example --config ./_example/config.yaml -- 1.txt
// FUCK_INT=111 go run ./_example --fuck-int=123 -- 1.txt
// FUCK_INT=111 go run ./_example --fuck-int=123 --env-file=./_example/.env -- 1.txt
