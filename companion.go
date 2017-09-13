package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"text/template"

	"github.com/golang/protobuf/proto"
	"gopkg.in/fsnotify.v1"
)

var cmd *exec.Cmd
var mk_is_running sync.Mutex

func watchForChanges(path string) {
	var watcher *fsnotify.Watcher
	var err error

	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("Cannot initialize fsnotify watcher: ",
			err.Error())
	}
	defer watcher.Close()

	err = watcher.Add(path)
	if err != nil {
		log.Fatal("Cannot start watching path ", path, ": ", err)
	}

	for {
		var event fsnotify.Event
		select {
		case event = <-watcher.Events:
			if event.Op == fsnotify.Create ||
				event.Op == fsnotify.Write {
				genFiles(path)
				updateBind()
			}
		case err = <-watcher.Errors:
			log.Print("Error watching ", path, ": ", err)
		}
	}
}

// Run make in the master zone directory upon change to regenerate files.
func genFiles(path string) {
	var mk *exec.Cmd
	var err error

	mk_is_running.Lock()
	defer mk_is_running.Unlock()

	mk = exec.Command("/usr/bin/make", "-C", path)
	mk.Stdin = os.Stdin
	mk.Stdout = os.Stdout
	mk.Stderr = os.Stderr

	err = mk.Run()
	if err != nil {
		log.Print("Cannot run make on ", path, ": ", err)
	}
}

// Tell bind to reload its configuration.
func updateBind() {
	if cmd != nil && cmd.Process != nil && cmd.ProcessState != nil &&
		!cmd.ProcessState.Exited() {
		// Send SIGHUP to the bind process.
		cmd.Process.Signal(syscall.SIGHUP)
	}
}

func runBind(bind_config, bind_user string) {
	var err error

	cmd = exec.Command("/usr/sbin/named", "-c", bind_config, "-g",
		"-u", bind_user, "-p", "5353")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	for {
		err = cmd.Run()
		if err != nil {
			// Exit the process on error.
			log.Print("Running named failed: ", err)
			return
		}
	}
}

func main() {
	var bind_config, bind_user, config_path string
	var bind_template_path string
	var watch_path string
	var file *os.File
	var tmpl *template.Template
	var config_bytes []byte
	var config BindConfig
	var err error

	flag.StringVar(&watch_path, "path", "/etc/bind/masterzones",
		"Path to watch for zone file changes")
	flag.StringVar(&bind_config, "bind-config", "/etc/bind/named.conf",
		"Full path of the named configuration file.")
	flag.StringVar(&bind_user, "bind-user", "named",
		"User to switch to when BIND started up.")
	flag.StringVar(&bind_template_path, "config-template",
		"/etc/bind/named.conf.tmpl",
		"Path of the bind configuration template.")
	flag.StringVar(&config_path, "config", "",
		"Configuration protocol buffer file with domain configs.")
	flag.Parse()

	config_bytes, err = ioutil.ReadFile(config_path)
	if err != nil {
		log.Fatal("Cannot read config file ", config_path, ": ", err)
	}

	err = proto.UnmarshalText(string(config_bytes), &config)
	if err != nil {
		log.Fatal("Error parsing config file ", config_path, ": ", err)
	}

	tmpl, err = template.ParseFiles(bind_template_path)
	if err != nil {
		log.Fatal("Cannot parse bind template file ",
			bind_template_path, ": ", err)
	}

	file, err = os.Create(bind_config)
	if err != nil {
		log.Fatal("Cannot open ", bind_config, " for writing: ", err)
	}

	err = tmpl.Execute(file, config)
	if err != nil {
		log.Fatal("Error applying bind template file ",
			bind_template_path, ": ", err)
	}

	err = file.Close()
	if err != nil {
		log.Fatal("Cannot close ", bind_config, " for writing: ", err)
	}

	genFiles(watch_path)
	go watchForChanges(watch_path)
	runBind(bind_config, bind_user)
}
