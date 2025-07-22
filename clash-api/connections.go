package clash_api

import (
	"net"
	"net/http"
	"time"

	c "github.com/daeuniverse/dae/control"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type MetaData struct {
	Host    string `json:"host"`
	Type    string `json:"type"`
	Network string `json:"network"`
}

type Connection struct {
	Id       string   `json:"id"`
	Metadata MetaData `json:"metadata"`
	Chains   []string `json:"chains"`
	Rule     string   `json:"rule"`
	Upload   int64    `json:"upload"`
	Download int64    `json:"download"`
}

type ConnectionInfo struct {
	Connections   []Connection `json:"connections"`
	DownloadTotal int64        `json:"downloadTotal"`
	UploadTotal   int64        `json:"uploadTotal"`
	Memory        int64        `json:"memory"`
}

func GetConnections(c *c.ControlPlane) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, "Failed to upgrade to WebSocket", http.StatusInternalServerError)
			return
		}
		defer conn.Close()

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			<-ticker.C
			connectionsMap := c.GetAllConnections()
			connectionsInfo := &ConnectionInfo{
				Connections:   make([]Connection, 0),
				DownloadTotal: 0,
				UploadTotal:   0,
				Memory:        0,
			}

			connectionsMap.Range(func(key, value any) bool {
				if conn, ok := value.(net.Conn); ok {
					connectionsInfo.Connections = append(connectionsInfo.Connections, Connection{
						Id: key.(string),
						Metadata: MetaData{
							Host:    conn.RemoteAddr().String(),
							Network: "tcp",
							Type:    "ebpf",
						},
						Chains:   []string{},
						Rule:     "",
						Upload:   0,
						Download: 0,
					})
				}
				return true
			})

			if err := conn.WriteJSON(connectionsInfo); err != nil {
				return
			}
		}
	}
}

func DeleteConnection(c *c.ControlPlane) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := c.AbortConnection(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func ConnetionRouter(c *c.ControlPlane) http.Handler {
	router := chi.NewRouter()
	router.Get("/", GetConnections(c))
	router.Delete("/{id}", DeleteConnection(c))

	return router
}
