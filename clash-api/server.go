package clash_api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/sirupsen/logrus"

	c "github.com/daeuniverse/dae/control"
)

func NewServer(c *c.ControlPlane, logger *logrus.Logger) {
	// 设置日志钩子
	SetupLogHook(logger)

	router := chi.NewRouter()

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	}))

	router.Mount("/connections", ConnetionRouter(c))
	router.Mount("/logs", LogsRouter(c))
	router.Get("/version", func(w http.ResponseWriter, r *http.Request) {
		version := map[string]string{
			"version": "1.0.0",
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(version)
	})

	http.ListenAndServe(":9098", router)
}
