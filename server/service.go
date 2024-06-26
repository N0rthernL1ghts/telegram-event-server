package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

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

var (
	clients   = make(map[*websocket.Conn]bool) // connected clients
	broadcast = make(chan []byte)
	upgrader  = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			allowedOriginsEnv := os.Getenv("ALLOWED_ORIGINS")
			if allowedOriginsEnv == "*" {
				log.Warn("Warning: Allowing all origins", zap.String("origin", origin))
				return true
			}
			allowedOrigins := strings.Split(allowedOriginsEnv, ",")
			for _, allowedOrigin := range allowedOrigins {
				if origin == allowedOrigin {
					return true
				}
			}
			log.Warn("Connection from disallowed origin", zap.String("origin", origin))
			return false
		},
	}
	log      *zap.Logger
	shutdown = make(chan struct{})
	wg       sync.WaitGroup
)

func main() {
	var err error
	log, err = zap.NewDevelopment(zap.IncreaseLevel(zapcore.InfoLevel), zap.AddStacktrace(zapcore.FatalLevel))
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	wg.Add(1)
	go handleMessages()

	http.HandleFunc("/events", handleConnections)

	go func() {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()
		if err := run(ctx); err != nil {
			panic(err)
		}
	}()

	server := &http.Server{Addr: ":8080"}

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			log.Fatal("ListenAndServe: ", zap.Error(err))
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	server.Shutdown(ctx)

	close(shutdown) // Close shutdown channel to signal goroutines to exit
	wg.Wait()       // Wait for all goroutines to exit

	log.Info("Interrupt signal received, shutting down...")
	os.Exit(0)
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal("Upgrade: ", zap.Error(err))
	}

	// Register our new client
	clients[ws] = true

	defer func() {
		delete(clients, ws)
		ws.Close()
	}()

	for {
		select {
		case <-shutdown:
			return
		default:
			_, _, err := ws.ReadMessage()
			if err != nil {
				return
			}
		}
	}
}

func handleMessages() {
	defer wg.Done()
	for {
		select {
		case <-shutdown:
			return
		case msg := <-broadcast:
			// Send it out to every client that is currently connected
			for client := range clients {
				err := client.WriteMessage(websocket.TextMessage, msg)
				if err != nil {
					log.Sugar().Errorf("error: %v", err)
					client.Close()
					delete(clients, client) // Ensure client is removed from the map
				}
			}
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

	// Setup message update handlers.
	d.OnNewChannelMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewChannelMessage) error {
		msg, _ := json.Marshal(update.Message)
		broadcast <- msg
		return nil
	})
	d.OnNewMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewMessage) error {
		msg, _ := json.Marshal(update.Message)
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
