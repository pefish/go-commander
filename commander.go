package commander

import (
	"flag"
	"fmt"
	"github.com/pefish/go-config"
	"github.com/pefish/go-logger"
	"github.com/pkg/errors"
	"io/ioutil"
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
	OnExited() error
}

type Commander struct {
	subcommands        map[string]ISubcommand
	version            string
	appName            string
	appDesc            string
	fnToSetCommonFlags func(flagSet *flag.FlagSet)

	data *StartData

	cacheFs *os.File
}

type StartData struct {
	DataDir    string
	LogLevel   string
	ConfigFile string
	SecretFile string
	Cache      []byte
}

func NewCommander(appName, version, appDesc string) *Commander {
	return &Commander{
		subcommands: make(map[string]ISubcommand),
		version:     version,
		appName:     appName,
		appDesc:     appDesc,
		data:        new(StartData),
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
		return errors.Errorf("subcommand error: %s", key)
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
	configFile := flagSet.String("config", os.Getenv("GO_CONFIG"), "path to config file")
	secretFile := flagSet.String("secret-file", os.Getenv("GO_SECRET"), "path to secret file")
	pprofEnable := flagSet.Bool("enable-pprof", false, "enable pprof")
	pprofAddress := flagSet.String("pprof-address", "0.0.0.0:9191", "<addr>:<port> to listen on for pprof")
	dataDir := flagSet.String("data-dir", os.ExpandEnv("$HOME/.")+commander.appName, "data dictionary")

	if commander.fnToSetCommonFlags != nil {
		commander.fnToSetCommonFlags(flagSet)
	}

	if subcommand != nil {
		err := subcommand.DecorateFlagSet(flagSet)
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

	dataDirStr := *dataDir
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
	commander.data.LogLevel = *logLevel
	commander.data.ConfigFile = *configFile
	commander.data.SecretFile = *secretFile

	err = go_config.Config.LoadConfig(go_config.Configuration{
		ConfigFilepath: *configFile,
		SecretFilepath: *secretFile,
	})
	if err != nil {
		return errors.Errorf("load config file error - %s", err)
	}
	go_config.Config.MergeFlagSet(flagSet)
	envKeyPairs := make(map[string]string, 5)
	for k, _ := range go_config.Config.Configs() {
		env := strings.ReplaceAll(strings.ToUpper(k), "-", "_")
		envKeyPairs[env] = k
	}
	go_config.Config.MergeEnvs(envKeyPairs)
	go_logger.Logger = go_logger.NewLogger(*logLevel)

	if *printVersion {
		fmt.Println(commander.version)
		os.Exit(0)
	}

	if subcommand == nil {
		return errors.Errorf("subcommand error: %s", key)
	}

	if pprofEnable != nil && *pprofEnable {
		pprofHttpServer := &http.Server{Addr: *pprofAddress}
		go func() { // 无需担心进程退出，不存在leak
			go_logger.Logger.InfoF("started pprof server on %s, you can open url [http://%s/debug/pprof/] to enjoy!!", pprofHttpServer.Addr, pprofHttpServer.Addr)
			err := pprofHttpServer.ListenAndServe()
			if err != nil {
				go_logger.Logger.WarnF("pprof server start error - %s", err)
			}
		}()
	}

	// load cache
	commander.cacheFs, err = os.OpenFile(path.Join(commander.data.DataDir, "data.json"), os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	b, err := ioutil.ReadAll(commander.cacheFs)
	if err != nil {
		return err
	}
	commander.data.Cache = b

	waitExit := make(chan error)
	go func() {
		waitExit <- subcommand.Start(commander.data)
	}()

	exitChan := make(chan os.Signal)
	signal.Notify(exitChan, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-exitChan:
		err := commander.onExited()
		if err != nil {
			go_logger.Logger.Error(errors.WithMessage(err, "commander OnExited failed"))
		}
		err = subcommand.OnExited()
		if err != nil {
			go_logger.Logger.Error(errors.WithMessage(err, "OnExited failed"))
		}
		return nil
	case result := <-waitExit:
		if result != nil {
			go_logger.Logger.Error(result)
		}
		err := commander.onExited()
		if err != nil {
			go_logger.Logger.Error(errors.WithMessage(err, "commander OnExited failed"))
		}
		err = subcommand.OnExited()
		if err != nil {
			go_logger.Logger.Error(errors.WithMessage(err, "OnExited failed"))
		}
		return result
	}
}

func (commander *Commander) onExited() error {
	if commander.data.Cache != nil {
		err := commander.cacheFs.Truncate(0)
		if err != nil {
			return err
		}
		_, err = commander.cacheFs.WriteAt(commander.data.Cache, 0)
		if err != nil {
			return err
		}
		err = commander.cacheFs.Sync()
		if err != nil {
			return err
		}
		err = commander.cacheFs.Close()
		if err != nil {
			return err
		}
	}

	return nil
}
