package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"

	"github.com/go-faster/errors"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/gotd/td/examples"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/updates"
	updhook "github.com/gotd/td/telegram/updates/hook"
	"github.com/gotd/td/tg"
)

type Client struct {
	conn *websocket.Conn
	send chan []byte
}

var (
	clients    = make(map[*Client]bool)
	broadcast  = make(chan []byte)
	register   = make(chan *Client)
	unregister = make(chan *Client)
	upgrader   = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			allowedOriginsEnv := os.Getenv("ALLOWED_ORIGINS")
			if allowedOriginsEnv == "*" {
				log.Warn("Warning: Allowing all origins", zap.String("origin", origin))
				return true
			}
			allowedOrigins := strings.Split(allowedOriginsEnv, ",")
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				if origin == allowedOrigin {
					allowed = true
					break
				}
			}
			if !allowed {
				log.Warn("Connection from disallowed origin", zap.String("origin", origin))
			}
			return allowed
		},
	}
	log           *zap.Logger
	clientMx      sync.Mutex
	activeClients sync.WaitGroup
)

func main() {
	var err error
	log, err = zap.NewDevelopment(zap.IncreaseLevel(zapcore.InfoLevel), zap.AddStacktrace(zapcore.FatalLevel))
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	go handleMessages()

	http.HandleFunc("/events", handleConnections)

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- http.ListenAndServe(":8080", nil)
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	go func() {
		if err := run(ctx); err != nil {
			log.Fatal("Failed to run: ", zap.Error(err))
			os.Exit(1)
		}
	}()

	select {
	case err := <-serverErr:
		if err != nil {
			log.Fatal("ListenAndServe: ", zap.Error(err))
		}
	case <-ctx.Done():
		log.Info("Interrupt signal received, shutting down...")
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("Upgrade: ", zap.Error(err))
		return
	}

	client := &Client{conn: ws, send: make(chan []byte)}
	register <- client

	go func() {
		defer func() {
			unregister <- client
			client.conn.Close()
		}()

		for {
			_, _, err := client.conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

func handleMessages() {
	for {
		select {
		case client := <-register:
			clientMx.Lock()
			clients[client] = true
			clientMx.Unlock()
			activeClients.Add(1)
		case client := <-unregister:
			clientMx.Lock()
			if _, ok := clients[client]; ok {
				delete(clients, client)
				close(client.send)
			}
			clientMx.Unlock()
			activeClients.Done()
		case message := <-broadcast:
			log.Info("Message received from broadcast", zap.ByteString("message", message))
			clientMx.Lock()
			for client := range clients {
				select {
				case client.send <- message:
				default:
					log.Warn("Failed to send message to client")
					delete(clients, client)
					close(client.send)
				}
			}
			clientMx.Unlock()
		}
	}
}

func run(ctx context.Context) error {
	d := tg.NewUpdateDispatcher()
	gaps := updates.New(updates.Config{
		Handler: d,
		Logger:  log.Named("gaps"),
	})

	// Authentication flow handles authentication process, like prompting for code and 2FA password.
	flow := auth.NewFlow(examples.Terminal{}, auth.SendCodeOptions{})

	// Initializing client from environment.
	// Available environment variables:
	// 	APP_ID:         app_id of Telegram app.
	// 	APP_HASH:       app_hash of Telegram app.
	// 	SESSION_FILE:   path to session file
	// 	SESSION_DIR:    path to session directory, if SESSION_FILE is not set
	client, err := telegram.ClientFromEnvironment(telegram.Options{
		Logger:        log,
		UpdateHandler: gaps,
		Middlewares: []telegram.Middleware{
			updhook.UpdateHook(gaps.Handle),
		},
	})
	if err != nil {
		return err
	}

	d.OnNewChannelMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewChannelMessage) error {
		msg, _ := json.Marshal(update.Message)
		log.Info("New channel message", zap.ByteString("message", msg))
		broadcast <- msg
		return nil
	})
	d.OnNewMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewMessage) error {
		msg, _ := json.Marshal(update.Message)
		log.Info("New message", zap.ByteString("message", msg))
		broadcast <- msg
		return nil
	})

	return client.Run(ctx, func(ctx context.Context) error {
		// Perform auth if no session is available.
		if err := client.Auth().IfNecessary(ctx, flow); err != nil {
			return errors.Wrap(err, "auth")
		}

		// Fetch user info.
		user, err := client.Self(ctx)
		if err != nil {
			return errors.Wrap(err, "call self")
		}

		return gaps.Run(ctx, client.API(), user.ID, updates.AuthOptions{
			OnStart: func(ctx context.Context) {
				log.Info("Gaps started")
			},
		})
	})
}
