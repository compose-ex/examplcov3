package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	app          = kingpin.New("examplcov3", "An etcd3 demonstration")
	endpointlist = app.Flag("endpoints", "etcd endpoints").Default("https://127.0.0.1:2379").OverrideDefaultFromEnvar("EX_ENDPOINTS").String()
	username     = app.Flag("user", "etcd User").OverrideDefaultFromEnvar("EX_USER").String()
	password     = app.Flag("pass", "etcd Password").OverrideDefaultFromEnvar("EX_PASS").String()
	config       = app.Command("config", "Change config data")
	configserver = config.Arg("server", "Server name").Required().String()
	configvar    = config.Arg("var", "Config variable").Required().String()
	configval    = config.Arg("val", "Config value").Required().String()
	server       = app.Command("server", "Go into server mode and listen for changes")
	servername   = server.Arg("server", "Server name").Required().String()
)

var configbase = "/config/"

func main() {
	kingpin.Version("0.0.1")
	command := kingpin.MustParse(app.Parse(os.Args[1:]))

	endpoints := strings.Split(*endpointlist, ",")

	cfg := clientv3.Config{
		Endpoints:   endpoints,
		Username:    *username,
		Password:    *password,
		DialTimeout: 5 * time.Second,
	}

	etcdclient, err := clientv3.New(cfg)

	if err != nil {
		log.Fatal(err)
	}

	defer etcdclient.Close()

	switch command {
	case config.FullCommand():
		doConfig(etcdclient)
	case server.FullCommand():
		doServer(etcdclient)
	}
}

func doConfig(etcdclient *clientv3.Client) {
	var key = configbase + *configserver + "/" + *configvar
	var val = *configval

	_, err := etcdclient.Put(context.TODO(), key, val)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s %s=%s\n", "put", key, val)
}

func doServer(etcd3client *clientv3.Client) {
	var key = configbase + *servername

	var settings map[string]string
	settings = make(map[string]string)

	resp, err := etcd3client.Get(context.TODO(), key, clientv3.WithPrefix())

	if err != nil {
		log.Fatal(err)
	}

	for _, ev := range resp.Kvs {
		_, setting := path.Split(string(ev.Key))
		settings[setting] = string(ev.Value)
	}

	fmt.Println(settings)

	watcher := etcd3client.Watch(context.TODO(), key, clientv3.WithPrefix())

	for resp := range watcher {
		for _, ev := range resp.Events {
			switch ev.Type {
			case mvccpb.PUT:
				_, setting := path.Split(string(ev.Kv.Key))
				settings[setting] = string(ev.Kv.Value)
			case mvccpb.DELETE:
				_, setting := path.Split(string(ev.Kv.Key))
				delete(settings, setting)
			}
		}
		fmt.Println(settings)
	}
}
