package commander

import (
	"flag"
	"fmt"
	"github.com/pefish/go-config"
	"github.com/pefish/go-logger"
	"os"
	"strings"
)

type ISubcommand interface {
	DecorateFlagSet(flagSet *flag.FlagSet) error
	// 启动子命令
	Start() error
}

type Commander struct {
	subcommands map[string]ISubcommand
	version     string
	appName     string
	appDesc     string
	fnToSetCommonFlags func(flagSet *flag.FlagSet)
}

func NewCommander(appName, version, appDesc string) *Commander {
	return &Commander{
		subcommands: make(map[string]ISubcommand),
		version:     version,
		appName:     appName,
		appDesc:     appDesc,
	}
}

func (commander *Commander) RegisterSubcommand(name string, subcommand ISubcommand) {
	commander.subcommands[name] = subcommand
}

// 没有指定子命令的时候，会执行这里注册的子命令
func (commander *Commander) RegisterDefaultSubcommand(subcommand ISubcommand) {
	commander.subcommands["default"] = subcommand
}

// 用于设置所有子命令共用的选项
func (commander *Commander) RegisterFnToSetCommonFlags(flagSet func(set *flag.FlagSet)) {
	commander.fnToSetCommonFlags = flagSet
}

func (commander *Commander) hasSubCommand() bool {
	return len(os.Args) != 1 && !strings.HasPrefix(os.Args[1], "-")
}

func (commander *Commander) Run() error {
	var subcommand ISubcommand
	key := "default"
	if commander.hasSubCommand() {
		key = os.Args[1]
	}
	subcommandTemp, ok := commander.subcommands[key]
	if ok {
		subcommand = subcommandTemp
	} else {
		return fmt.Errorf("subcommand error: %s", key)
	}

	flagSet := flag.NewFlagSet(commander.appName, flag.ExitOnError)

	flagSet.Usage = func() {
		fmt.Printf("\n%s\n\n", commander.appDesc)
		fmt.Printf("Usage of %s:\n", flagSet.Name())
		flagSet.PrintDefaults()
		fmt.Printf("\n")
	}
	printVersion := flagSet.Bool("version", false, "print version string")
	logLevel := flagSet.String("log-level", "info", "set log verbosity: debug, info, warn, or error")
	configFile := flagSet.String("config", "", "path to config file")

	if commander.fnToSetCommonFlags != nil {
		commander.fnToSetCommonFlags(flagSet)
	}

	if subcommand != nil {
		err := subcommand.DecorateFlagSet(flagSet)
		if err != nil {
			return err
		}
	}

	argsToParse := os.Args[1:]
	if commander.hasSubCommand() {
		argsToParse = os.Args[2:]
	}
	err := flagSet.Parse(argsToParse)
	if err != nil {
		return err
	}

	if configFile != nil && *configFile != "" {
		err := go_config.Config.LoadYamlConfig(go_config.Configuration{
			ConfigFilepath: *configFile,
		})
		if err != nil {
			return fmt.Errorf("load config file error - %s", err)
		}
	}
	go_config.Config.MergeFlagSet(flagSet)

	go_logger.Logger = go_logger.NewLogger(*logLevel)

	if *printVersion {
		fmt.Println(commander.version)
		os.Exit(0)
	}

	if subcommand != nil {
		err := subcommand.Start()
		if err != nil {
			go_logger.Logger.Error(err)
			return nil
		}
	} else {
		return fmt.Errorf("subcommand error: %s", key)
	}
	return nil
}
