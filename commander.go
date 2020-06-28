package commander

import (
	"flag"
	"fmt"
	"github.com/pefish/go-config"
	"github.com/pefish/go-logger"
	"log"
	"os"
)

type SubcommandInterface interface {
	DecorateFlagSet(flagSet *flag.FlagSet) error
	ParseFlagSet(flagset *flag.FlagSet) error
	// 启动子命令
	Start() error
}

type Commander struct {
	subcommands map[string]SubcommandInterface
	version     string
	appName     string
	appDesc     string
}

func NewCommander(appName, version, appDesc string) *Commander {
	return &Commander{
		subcommands: make(map[string]SubcommandInterface),
		version:     version,
		appName:     appName,
		appDesc:     appDesc,
	}
}

func (commander *Commander) RegisterSubcommand(name string, subcommand SubcommandInterface) {
	commander.subcommands[name] = subcommand
}

func (commander *Commander) Run() {
	var subcommand SubcommandInterface
	if len(os.Args) < 2 {
		fmt.Println("参数错误")
		return
	}
	secondArg := os.Args[1]
	subcommandTemp, ok := commander.subcommands[secondArg]
	if ok {
		subcommand = subcommandTemp
	}

	flagSet := flag.NewFlagSet(commander.appName, flag.ExitOnError)

	flagSet.Usage = func() {
		fmt.Fprintf(flagSet.Output(), "\n%s\n\n", commander.appDesc)
		fmt.Fprintf(flagSet.Output(), "Usage of %s:\n", flagSet.Name())
		flagSet.PrintDefaults()
		fmt.Fprintf(flagSet.Output(), "\n")
	}
	flagSet.Bool("version", false, "print version string")
	flagSet.String("log-level", "info", "set log verbosity: debug, info, warn, or error")
	flagSet.String("config", "", "path to config file")

	if subcommand != nil {
		err := subcommand.DecorateFlagSet(flagSet)
		if err != nil {
			log.Fatal(err)
		}
		err = subcommand.ParseFlagSet(flagSet)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		err := flagSet.Parse(os.Args[1:])
		if err != nil {
			log.Fatal(err)
		}
	}

	configFile := flagSet.Lookup("config").Value.(flag.Getter).Get().(string)
	err := go_config.Config.LoadYamlConfig(go_config.Configuration{
		ConfigFilepath: configFile,
	})
	if err != nil {
		log.Fatal(fmt.Errorf("load config file error - %s", err))
	}
	go_config.Config.MergeFlagSet(flagSet)

	logLevel, err := go_config.Config.GetString("log-level")
	if err != nil {
		log.Fatal(err)
	}
	go_logger.Logger = go_logger.NewLogger(logLevel)

	printVersion, err := go_config.Config.GetBool("version")
	if err != nil {
		log.Fatal(err)
	}
	if printVersion {
		fmt.Println(commander.version)
		os.Exit(0)
	}

	if subcommand != nil {
		err := subcommand.Start()
		if err != nil {
			go_logger.Logger.Error(err)
			return
		}
	}
}
