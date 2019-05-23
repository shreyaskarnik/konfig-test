package main

import (
	"fmt"
	"log"
	"os"

	"github.com/lalamove/konfig/loader/klconsul"
	"github.com/lalamove/nui/nlogger"

	"time"

	"encoding/json"
	"net/http"

	consul "github.com/hashicorp/consul/api"
	vault "github.com/hashicorp/vault/api"
	"github.com/lalamove/konfig"
	"github.com/lalamove/konfig/loader/klfile"

	"github.com/lalamove/konfig/loader/klvault"
	"github.com/lalamove/konfig/loader/klvault/auth/token"
	"github.com/lalamove/konfig/parser/kptoml"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Creds contains credentials and secrets
type Creds struct {
	ConsulToken string `konfig:"token" json:"token"`
	ApiKey      string `konfig:"api.key" json:"api_key"`
	SecretV2    string `konfig:"my-value"`
	SecretV1    string `konfig:"my-value1"`
}

var config Creds

func main() {
	loP := nlogger.New(os.Stdout, "CONFIG | ")
	logger := nlogger.NewProvider(loP)
	http.Handle("/metrics", promhttp.Handler())
	konfig.Init(&konfig.Config{
		Metrics: true,
		Name:    "root",
	})
	vaultClient, err := vault.NewClient(&vault.Config{
		Address: "http://0.0.0.0:8200",
	})
	if err != nil {
		fmt.Println(err)
	}
	vaultLoader := klvault.New(&klvault.Config{
		Secrets: []klvault.Secret{
			{
				Key: "/consul/creds/myrole",
			},
			// breaks right now
			{
				Key: "/secret/data/my-secret?version=2",
			},
			{
				Key: "/secretv1/my-secret",
			},
			// uncomment to test version 1
			// {
			// 	Key: "/secret/data/my-secret?version=1",
			// },
		},
		Client: vaultClient,
		AuthProvider: &token.Token{
			T: "swordfish",
		},
		Renew: true,
		Debug: true,
	})
	fileLoader := klfile.New(&klfile.Config{
		Files: []klfile.File{
			{
				Path:   "./config.toml",
				Parser: kptoml.Parser,
			},
		},
		Watch: true,
		Rate:  1 * time.Second, // Rate for the polling watching the file changes
		Debug: false,
	})
	konfig.Bind(Creds{})
	config = konfig.Value().(Creds)
	konfig.RegisterLoaderWatcher(fileLoader, reload)
	konfig.RegisterLoaderWatcher(vaultLoader, reload)
	if err := konfig.LoadWatch(); err != nil {
		log.Fatal(err)
	}
	consulConfig := consul.DefaultConfig()
	consulConfig.Address = "http://0.0.0.0:8500"
	consulConfig.Token = konfig.MustString("token")
	consulClient, err := consul.NewClient(consulConfig)
	if err != nil {
		fmt.Println(err)
	}
	consulLoader := klconsul.New(&klconsul.Config{
		StrictMode:    false,
		Debug:         true,
		Watch:         true,
		StopOnFailure: false,
		Client:        consulClient,
		Keys: []klconsul.Key{
			{
				Key:    "test/asdf/test/toml",
				Parser: kptoml.Parser,
			},
		},
		Logger: logger,
	})
	konfig.RegisterLoaderWatcher(consulLoader, reload)
	if err := konfig.LoadWatch(); err != nil {
		log.Fatal(err)
	}
	config = konfig.Value().(Creds)
	http.HandleFunc("/info", config.infoHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
func (i *Creds) infoHandler(responseWriter http.ResponseWriter, request *http.Request) {
	js, err := json.Marshal(i)
	if err != nil {
		http.Error(responseWriter, err.Error(), http.StatusInternalServerError)
		return
	}

	responseWriter.Header().Set("Content-Type", "application/json")
	_, err = responseWriter.Write(js)
	if err != nil {
		log.Println("cannot write response for info handler", err)
	}

}

func reload(c konfig.Store) error {
	config = konfig.Value().(Creds)
	return nil
}
