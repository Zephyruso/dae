package clash_api

import (
	"net/http"
	"sync"

	c "github.com/daeuniverse/dae/control"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type LogEntry struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

type LogBroadcaster struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan LogEntry
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.RWMutex
}

func NewLogBroadcaster() *LogBroadcaster {
	return &LogBroadcaster{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan LogEntry, 100),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

func (lb *LogBroadcaster) Start() {
	for {
		select {
		case client := <-lb.register:
			lb.mu.Lock()
			lb.clients[client] = true
			lb.mu.Unlock()

		case client := <-lb.unregister:
			lb.mu.Lock()
			if _, ok := lb.clients[client]; ok {
				delete(lb.clients, client)
				client.Close()
			}
			lb.mu.Unlock()

		case logEntry := <-lb.broadcast:
			lb.mu.RLock()
			for client := range lb.clients {
				err := client.WriteJSON(logEntry)
				if err != nil {
					client.Close()
					delete(lb.clients, client)
				}
			}
			lb.mu.RUnlock()
		}
	}
}

func (lb *LogBroadcaster) Broadcast(entry LogEntry) {
	select {
	case lb.broadcast <- entry:
	default:
	}
}

var globalLogBroadcaster *LogBroadcaster

func init() {
	globalLogBroadcaster = NewLogBroadcaster()
	go globalLogBroadcaster.Start()
}

type LogHook struct {
	broadcaster *LogBroadcaster
}

func NewLogHook(broadcaster *LogBroadcaster) *LogHook {
	return &LogHook{
		broadcaster: broadcaster,
	}
}

func (h *LogHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *LogHook) Fire(entry *logrus.Entry) error {
	logEntry := LogEntry{
		Type:    entry.Level.String(),
		Payload: entry.Message,
	}

	h.broadcaster.Broadcast(logEntry)
	return nil
}

func GetLogs(c *c.ControlPlane) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, "Failed to upgrade to WebSocket", http.StatusInternalServerError)
			return
		}

		globalLogBroadcaster.register <- conn

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}

		globalLogBroadcaster.unregister <- conn
	}
}

func LogsRouter(c *c.ControlPlane) http.Handler {
	router := chi.NewRouter()
	router.Get("/", GetLogs(c))
	return router
}

func SetupLogHook(logger *logrus.Logger) {
	if logger != nil {
		hook := NewLogHook(globalLogBroadcaster)
		logger.AddHook(hook)
	}
}
