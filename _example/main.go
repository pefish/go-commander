package main

import (
	_ "net/http/pprof"
	"sync"

	commander2 "github.com/pefish/go-commander"
	"github.com/pefish/go-commander/pkg/persistence"
	go_config "github.com/pefish/go-config"
)

type Config struct {
	Test                   string `json:"test" default:"default-flag-test" usage:"test flag set"`
	FuckInt                int    `json:"fuck-int" default:"888" usage:"test fuck int"`
	Abc                    int    `json:"abc" default:"888" usage:"test fuck int"`
	commander2.BasicConfig `json:",omitempty"`
}

var config struct {
	Config
	Name  string `json:"name" default:"name" usage:"Name."`
	Name1 string `json:"name1" usage:"Name1."`
}

var Data struct {
	Data1 string `json:"data1"`
}

type TestSubCommand struct {
}

func (t TestSubCommand) Config() interface{} {
	return &config
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
	command.Logger.InfoFRaw("config: %#v", config)
	command.Logger.InfoFRaw("Data: %#v", Data)
	Data.Data1 = "data2"

	// var a sync.Map
	// a.Store(11, "aa")
	// err := persistence.SaveToDisk("./a.glob", &a)
	// if err != nil {
	// 	return err
	// }

	// var b sync.Map
	// err = persistence.LoadFromDisk("./a.glob", &b)
	// if err != nil {
	// 	return err
	// }
	// b.Range(func(key, value any) bool {
	// 	command.Logger.InfoF("<key: %#v> <value: %#v>", key, value)
	// 	return true
	// })
	return nil
}

func (t TestSubCommand) OnExited(data *commander2.Commander) error {
	//fmt.Println("OnExited")
	return nil
}

func main() {
	//go_logger.Logger.Error(errors.WithMessage(errors.New("123"), "ywrtsdfhs"))
	commander := commander2.New("test", "v0.0.1", "小工具")
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
		commander.Logger.Error(err)
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
// FUCK_INT=111 go run ./_example --fuck-int=123 --env-file=./_example/.env --name=name1 -- 1.txt
