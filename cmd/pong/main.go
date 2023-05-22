package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/kelseyhightower/envconfig"
)

type appConfig struct {
	HostAddr string
	Port     uint16 `default:"80"`
}

type pingHandler struct{}

func (*pingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/ping" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Add("content-type", "text/plain")
	w.Write([]byte("pong"))

	log.Printf("Received ping from %s", r.RemoteAddr)
}

func main() {
	config := appConfig{}
	if err := envconfig.Process("PONG", &config); err != nil {
		log.Fatalf("Error processing env config: %s", err)
	}

	if err := createNomadPingJob(config.HostAddr); err != nil {
		log.Fatalf("Error creating Ping job in Nomad: %s", err)
	}

	server := http.Server{
		Addr:    fmt.Sprintf(":%d", config.Port),
		Handler: &pingHandler{},
	}

	quitChan := make(chan struct{}, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func(s *http.Server, w *sync.WaitGroup, qc <-chan struct{}) {
		defer w.Done()

		sc := make(chan os.Signal, 1)
		signal.Notify(sc, os.Interrupt, syscall.SIGTERM)

		select {
		case <-qc:
		case <-sc:
			log.Print("Received OS interrupt or kill signal. Shutting down server.")
			err := server.Shutdown(context.Background())
			if err != nil {
				log.Printf("Error shutting server down: %s", err)
			}
		}
	}(&server, &wg, quitChan)

	log.Print("Starting server at " + server.Addr)
	err := server.ListenAndServe()
	close(quitChan)
	wg.Wait()

	if err != nil {
		log.Fatal(err)
	}
}

func createNomadPingJob(targetAddr string) error {
	var (
		ping             = "ping"
		jobType          = "batch"
		groupCount       = 1
		taskDriver       = "docker"
		configImage      = "ping:0.0.1"
		resourceCPU      = 10
		resourceMemoryMB = 10
		restartAttempts  = 3
	)

	time.Sleep(5 * time.Second)

	config := api.DefaultConfig()
	config.Address = "http://nomad:4646"
	client, err := api.NewClient(config)
	if err != nil {
		return fmt.Errorf("error creating Nomad client: %w", err)
	}

	job := &api.Job{
		ID:   &ping,
		Type: &jobType,

		TaskGroups: []*api.TaskGroup{
			{
				Name:  &ping,
				Count: &groupCount,

				Tasks: []*api.Task{
					{
						Name:   ping,
						Driver: taskDriver,
						Config: map[string]interface{}{
							// Nomad Docker Driver cannot specify docker network to join.
							// Hence we need to supply docker host ip address here.
							"args":  []string{targetAddr},
							"image": &configImage,
						},
						Resources: &api.Resources{
							CPU:      &resourceCPU,
							MemoryMB: &resourceMemoryMB,
						},
						RestartPolicy: &api.RestartPolicy{
							Attempts: &restartAttempts,
						},
					},
				},
			},
		},
	}

	_, _, err = client.Jobs().Validate(job, nil)
	if err != nil {
		return fmt.Errorf("error from job validation: %w", err)
	}

	_, _, err = client.Jobs().Register(job, nil)
	if err != nil {
		return fmt.Errorf("error registering Ping job to Nomad: %w", err)
	}

	return nil
}
