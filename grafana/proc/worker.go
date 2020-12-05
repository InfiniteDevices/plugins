package proc

import (
	"log"

	"github.com/InfiniteDevices/plugins/grafana/api"
	redistimeseries "github.com/RedisTimeSeries/redistimeseries-go"
)

type objectWorker struct {
	obj         api.Object
	api         api.Handler
	done        chan struct{}
	redisClient *redistimeseries.Client
}

func (w *objectWorker) Start() {
	createOpts := redistimeseries.CreateOptions{
		Labels: map[string]string{
			"uid":  w.obj.UID,
			"name": w.obj.Name,
			"kind": w.obj.Kind,
		},
	}

	ch, err := w.api.GetDevicesStateStream(w.obj.UID)
	if err != nil {
		log.Printf("error on get devices state stream: %s\n", err)
	}

	createdKeys := map[string]bool{}

	for {
		select {
		case <-w.done:
			return
		case state := <-ch:
			if state == nil {
				log.Printf("received nil state for object %v", w.obj)
				continue
			}
			for k, v := range state.Result.ReportedState.Data {
				if v == nil {
					continue
				}
				f, ok := v.(float64)
				if !ok {
					log.Printf("invalid data type found for object %v and key %s", w.obj, k)
					continue
				}
				if !createdKeys[k] {
					_ = w.redisClient.CreateKeyWithOptions(k+":"+w.obj.UID, createOpts)
					createdKeys[k] = true
				}
				_, err = w.redisClient.AddAutoTsWithOptions(k+":"+w.obj.UID, f, createOpts)
				if err != nil {
					log.Printf("failed to add time series item: %s\n", err)
					continue
				}
				log.Printf("added time series item: object=%v key=%s\n", w.obj, k)
			}
		}
	}
}

func (w *objectWorker) Stop() {
	close(w.done)
}
