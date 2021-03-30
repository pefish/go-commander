package commander

import (
	"context"
	"flag"
	"fmt"
	go_config "github.com/pefish/go-config"
	"github.com/pefish/go-logger"
	"github.com/pkg/errors"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
)

type ISubcommand interface {
	DecorateFlagSet(flagSet *flag.FlagSet) error
	// 启动子命令
	Start(data *StartData) error
	// 用于优雅退出
	OnExited(data *StartData) error
}

type SubcommandInfo struct{
	desc string
	subcommand ISubcommand
}

type Commander struct {
	subcommands        map[string]*SubcommandInfo
	version            string
	appName            string
	appDesc            string
	fnToSetCommonFlags func(flagSet *flag.FlagSet)

	data *StartData

	cacheFs *os.File

	subCommandValid bool

	cancelFuncOfExitCancelCtx  context.CancelFunc
}

type StartData struct {
	DataDir    string
	LogLevel   string
	ConfigFile string
	SecretFile string
	Cache      Cache

	Args []string
	ExitCancelCtx context.Context
}

func NewCommander(appName, version, appDesc string) *Commander {
	return &Commander{
		subcommands: make(map[string]*SubcommandInfo),
		version:     version,
		appName:     appName,
		appDesc:     appDesc,
		data:        new(StartData),
		subCommandValid: true,
	}
}

func (commander *Commander) RegisterSubcommand(name string, desc string, subcommand ISubcommand) {
	commander.subcommands[name] = &SubcommandInfo{
		desc:       desc,
		subcommand: subcommand,
	}
}

// 没有指定子命令的时候，会执行这里注册的子命令
func (commander *Commander) RegisterDefaultSubcommand(desc string, subcommand ISubcommand) {
	commander.subcommands["default"] = &SubcommandInfo{
		desc:       desc,
		subcommand: subcommand,
	}
}

// 用于设置所有子命令共用的选项
func (commander *Commander) RegisterFnToSetCommonFlags(flagSet func(set *flag.FlagSet)) {
	commander.fnToSetCommonFlags = flagSet
}

func (commander *Commander) DisableSubCommand() {
	commander.subCommandValid = false
}

func (commander *Commander) hasSubCommand() bool {
	return len(os.Args) != 1 && !strings.HasPrefix(os.Args[1], "-") && commander.subCommandValid
}

func (commander *Commander) Run() error {
	key := "default"
	if commander.hasSubCommand() {
		key = os.Args[1]
	}
	subcommandInfo, ok := commander.subcommands[key]
	if !ok {
		return errors.Errorf("subcommand error: %s", key)
	}

	flagSet := flag.NewFlagSet(commander.appName, flag.ExitOnError)
	flagSet.Usage = func() {

		fmt.Printf("\n%s\n\n", commander.appDesc)

		fmt.Printf("Usage of <%s>:\n", flagSet.Name())
		fmt.Printf("  %s [subcommand] [options] [args]\n\n", flagSet.Name())

		// 如果有子命令，打印所有子命令
		if commander.subCommandValid {
			if len(commander.subcommands) > 0 {
				fmt.Println("Commands:")
				for name, info := range commander.subcommands {
					if name == "default" {
						fmt.Printf("  [default]\tdefault subcommand. %s\n", info.desc)
					} else {
						fmt.Printf("  %s\t%s\n", name, info.desc)
					}
				}
				fmt.Printf("\n")
			}
		}

		fmt.Println("Options:")
		flagSet.PrintDefaults()
		fmt.Printf("\n")
	}
	flagSet.Bool("version", false, "print version string")
	flagSet.String("log-level", "info", "set log verbosity: debug, info, warn, or error")
	configFile := flagSet.String("config", os.Getenv("GO_CONFIG"), "path to config file")
	secretFile := flagSet.String("secret-file", os.Getenv("GO_SECRET"), "path to secret file")
	flagSet.Bool("enable-pprof", false, "enable pprof")
	flagSet.String("pprof-address", "0.0.0.0:9191", "<addr>:<port> to listen on for pprof")
	flagSet.String("data-dir", os.ExpandEnv("$HOME/.")+commander.appName, "data dictionary")
	if commander.fnToSetCommonFlags != nil {
		commander.fnToSetCommonFlags(flagSet)
	}
	if subcommandInfo != nil {
		err := subcommandInfo.subcommand.DecorateFlagSet(flagSet)
		if err != nil {
			return errors.Wrap(err, "decorate flagSet error")
		}
	}
	argsToParse := os.Args[1:]
	if commander.hasSubCommand() {
		argsToParse = os.Args[2:]
	}
	err := flagSet.Parse(argsToParse)
	if err != nil {
		return errors.Wrap(err, "parse flagSet error")
	}

	// merge envs and config file
	err = go_config.ConfigManagerInstance.LoadConfig(go_config.Configuration{
		ConfigFilepath: *configFile,
		SecretFilepath: *secretFile,
	})
	if err != nil {
		return errors.Errorf("load config file error - %s", err)
	}
	go_config.ConfigManagerInstance.MergeFlagSet(flagSet)
	envKeyPairs := make(map[string]string, 5)
	for k, _ := range go_config.ConfigManagerInstance.Configs() {
		env := strings.ReplaceAll(strings.ToUpper(k), "-", "_")
		envKeyPairs[env] = k
	}
	go_config.ConfigManagerInstance.MergeEnvs(envKeyPairs)


	dataDirStr, err := go_config.ConfigManagerInstance.GetString("data-dir")
	if err != nil {
		return errors.Wrap(err, "get data-dir config error")
	}
	fsStat, err := os.Stat(dataDirStr)
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") || fsStat == nil || !fsStat.IsDir() {
			err = os.Mkdir(dataDirStr, 0755)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	commander.data.DataDir = dataDirStr
	logLevel, err := go_config.ConfigManagerInstance.GetString("log-level")
	if err != nil {
		return errors.Wrap(err, "get log-level config error")
	}
	commander.data.LogLevel = logLevel
	commander.data.ConfigFile = *configFile
	commander.data.SecretFile = *secretFile
	commander.data.Args = flagSet.Args()

	ctx, cancel := context.WithCancel(context.Background())
	commander.data.ExitCancelCtx = ctx
	commander.cancelFuncOfExitCancelCtx = cancel

	go_logger.Logger = go_logger.NewLogger(logLevel)

	printVersion, err := go_config.ConfigManagerInstance.GetBool("version")
	if err != nil {
		return errors.Wrap(err, "get version config error")
	}
	if printVersion {
		fmt.Println(commander.version)
		os.Exit(0)
	}

	if subcommandInfo == nil {
		return errors.Errorf("subcommand error: %s", key)
	}

	pprofEnable, err := go_config.ConfigManagerInstance.GetBool("enable-pprof")
	if err != nil {
		return errors.Wrap(err, "get enable-pprof config error")
	}
	pprofAddress, err := go_config.ConfigManagerInstance.GetString("pprof-address")
	if err != nil {
		return errors.Wrap(err, "get version config error")
	}
	if pprofEnable {
		pprofHttpServer := &http.Server{Addr: pprofAddress}
		go func() { // 无需担心进程退出，不存在leak
			go_logger.Logger.InfoF("started pprof server on %s, you can open url [http://%s/debug/pprof/] to enjoy!!", pprofHttpServer.Addr, pprofHttpServer.Addr)
			err := pprofHttpServer.ListenAndServe()
			if err != nil {
				go_logger.Logger.WarnF("pprof server start error - %s", err)
			}
		}()
	}

	// load cache
	err = commander.data.Cache.Init(path.Join(commander.data.DataDir, "data.json"))
	if err != nil {
		return err
	}

	waitErrorChan := make(chan error)
	go func() {
		waitErrorChan <- subcommandInfo.subcommand.Start(commander.data)
	}()

	exitChan := make(chan os.Signal)
	signal.Notify(exitChan, syscall.SIGINT, syscall.SIGTERM)

	var exitErr error

	ctrlCCount := 3
	ctrlCCountTemp := ctrlCCount
forceExit:
	for {
		select {
		case <-exitChan:
			// 要等待 start 函数退出
			if ctrlCCountTemp == ctrlCCount {
				commander.cancelFuncOfExitCancelCtx()  // 通知下去，程序即将退出
			}
			ctrlCCountTemp--
			go_logger.Logger.InfoF("Got interrupt, exiting... %d", ctrlCCountTemp)
			if ctrlCCountTemp <= 0 {  // Ctrl C n 次强制退出，不等 start 函数了
				break forceExit
			}
			break
		case exitErr = <-waitErrorChan:
			break forceExit
		}
	}


	err = commander.onExitedBefore()
	if err != nil {
		exitErr = errors.WithMessage(exitErr, fmt.Sprintf("commander OnExitedBefore failed - %s", err.Error()))
	}
	err = subcommandInfo.subcommand.OnExited(commander.data)
	if err != nil {
		exitErr = errors.WithMessage(exitErr, fmt.Sprintf("OnExited failed - %s", err.Error()))
	}
	err = commander.onExitedAfter()
	if err != nil {
		exitErr = errors.WithMessage(exitErr, fmt.Sprintf("commander OnExitedAfter failed - %s", err.Error()))
	}
	return exitErr
}

func (commander *Commander) onExitedAfter() error {
	return nil
}

func (commander *Commander) onExitedBefore() error {
	return nil
}
