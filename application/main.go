package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"service-discovery/shared"
	"time"

	"github.com/labstack/echo"
)

var (
	port      string
	startTime time.Time
	forceLag  bool
)

func main() {
	// Assign custom port, with default value 8080
	flag.StringVar(&port, "port", "8081", "assign custom port")
	flag.Parse()

	// Register server to discovery service
	err := registerServer("http://localhost:" + port)
	if err != nil {
		log.Fatal(err.Error())
	}

	e := echo.New()
	// Registered routes
	e.GET("/healthcheck", healthcheckHandler)
	e.POST("/transaction", transactionHandler)
	e.POST("/force-lag/true", func(ctx echo.Context) error {
		forceLag = true
		return nil
	})
	e.POST("/force-lag/false", func(ctx echo.Context) error {
		forceLag = false
		return nil
	})

	// Start the server
	startTime = time.Now()
	log.Printf("Starting server on %s :...", port)
	if err := e.Start(fmt.Sprint(":", port)); err != nil {
		log.Fatal(err)
	}
}

// Handlers
func healthcheckHandler(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK,
		map[string]interface{}{
			"uptime": time.Since(startTime).String(),
		},
	)
}

type PointTransaction struct {
	Game    string `json:"game"`
	GamerID string `json:"gamer_id"`
	Points  int    `json:"points"`
}

func transactionHandler(ctx echo.Context) error {
	transactionData := &PointTransaction{}
	if ctx.QueryParam("is_force_lag") == "true" || forceLag {
		time.Sleep(1 * time.Second)
	}
	err := ctx.Bind(transactionData)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": err.Error(),
		})
	}
	return ctx.JSON(http.StatusOK, transactionData)
}

func registerServer(url string) error {
	newServer := &shared.NewServer{
		URL: url,
	}
	b, err := json.Marshal(newServer)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", "http://localhost:8888/register", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New("service registration not successful")
	}
	return nil
}