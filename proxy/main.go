package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"service-discovery/shared"
	"sync"
	"time"

	"github.com/labstack/echo"
)

func main() {
	// Initiate load balancer handler
	lb := NewLoadBalancer()

	// Initiate healthcheck ticker
	go lb.healthCheck()
	e := echo.New()

	// Registered routes
	e.GET("/stats", lb.Stats)
	e.POST("/register", lb.RegisterServer)
	e.Any("/*", lb.ProxyHandler)

	// Start the server
	log.Printf("Starting server on :8888 ...")
	if err := e.Start(":8888"); err != nil {
		log.Fatal(err)
	}
}

// Server hold metadata of each instance
type ServerStatus string

const (
	ServerStatusHEALTHY   ServerStatus = "HEALTHY"   // Server run normally
	ServerStatusUNHEALTHY ServerStatus = "UNHEALTHY" // Server request error 3 times in a row
	ServerStatusDOWN      ServerStatus = "DOWN"      // Server not passing healthcheck 3 times in a row

	RequestErrorThreshold     = 3
	HealthcheckErrorThreshold = 3
)

type Server struct {
	sync.Mutex
	URL                   string
	HitCount              int
	HealthCheckErrorCount int
	RequestErrorCount     int
	Status                ServerStatus
	RecoverTime           time.Time
}

// Handle if request to application instance is success
func (s *Server) handleRequestSuccess() {
	s.Lock()
	defer s.Unlock()
	s.RequestErrorCount = 0
	s.Status = ServerStatusHEALTHY
}

// Handle if request to application instance is error
func (s *Server) handleRequestError() {
	s.Lock()
	defer s.Unlock()
	s.RequestErrorCount += 1
	if s.RequestErrorCount >= RequestErrorThreshold && s.Status == ServerStatusHEALTHY {
		s.Status = ServerStatusUNHEALTHY
		s.RecoverTime = time.Now().Add(30 * time.Second)
	}
}

// Handle if health check is success
func (s *Server) handleHealthCheckSuccess() {
	s.Lock()
	defer s.Unlock()
	if s.Status == ServerStatusDOWN {
		s.Status = ServerStatusHEALTHY
		s.HealthCheckErrorCount = 0
	}
}

// Handle if health check is error
func (s *Server) handleHealthCheckError() {
	s.Lock()
	defer s.Unlock()
	s.HealthCheckErrorCount += 1
	if s.HealthCheckErrorCount >= HealthcheckErrorThreshold {
		s.Status = ServerStatusDOWN
	}
}

// Load balancer need collection of servers
type LoadBalancer struct {
	sync.Mutex
	Client   http.Client
	Servers  []*Server
	Counter  int
	TotalHit int
}

func NewLoadBalancer() *LoadBalancer {
	return &LoadBalancer{
		Servers: []*Server{},
		Client: http.Client{
			Timeout: 100 * time.Millisecond,
		},
	}
}

// ProxyHandler handle client request and pass it to application instance
func (lb *LoadBalancer) ProxyHandler(ctx echo.Context) error {
	if len(lb.Servers) == 0 {
		return echo.NewHTTPError(http.StatusRequestTimeout, "No healthy server")
	}
	// Select the server
	server := lb.selectServer()

	// Construct and call to the selected server
	reqUrl := fmt.Sprintf("%s%s?%s", server.URL, ctx.Request().URL.Path, ctx.Request().URL.RawQuery)
	req, err := http.NewRequest(ctx.Request().Method, reqUrl, ctx.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct request")
	}
	req.Header = ctx.Request().Header
	// req.Body = ctx.Request().Body
	resp, err := lb.Client.Do(req)
	if err != nil {
		// Check if the error is a timeout
		server.handleRequestError()
		if urlErr, ok := err.(*url.Error); ok && urlErr.Timeout() {
			return echo.NewHTTPError(http.StatusRequestTimeout, "Request to backend server timed out")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to reach backend server")
	}
	defer resp.Body.Close()
	server.handleRequestSuccess()
	// Copy the response from the backend server to the original client
	for key, values := range resp.Header {
		for _, value := range values {
			ctx.Response().Header().Add(key, value)
		}
	}
	ctx.Response().WriteHeader(resp.StatusCode)
	_, err = io.Copy(ctx.Response().Writer, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// selectServer choose a server using round robin algorithm
func (lb *LoadBalancer) selectServer() *Server {
	// Lock to handle concurrent increment
	lb.Lock()
	defer lb.Unlock()

	// Select the server based on the current state
	server := lb.Servers[lb.Counter%len(lb.Servers)]
	// Reselect server if it's down or unhealthy but still in recover period
	for server.Status == ServerStatusDOWN || (server.Status == ServerStatusUNHEALTHY && time.Now().Before(server.RecoverTime)) {
		lb.Counter++
		server = lb.Servers[lb.Counter%len(lb.Servers)]
	}
	server.HitCount += 1
	lb.TotalHit += 1
	// Increment the counter for next request to get the next service
	lb.Counter += 1
	fmt.Printf("hit server: %s, count: %d \n", server.URL, server.HitCount)
	return server
}

// RegisterServer will be triggered from application instance when initiated and register the metadata to the loadbalancer
func (lb *LoadBalancer) RegisterServer(ctx echo.Context) error {
	newServer := &shared.NewServer{}
	if err := ctx.Bind(newServer); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to bind request")
	}
	lb.assignServer(newServer)
	ctx.Response().WriteHeader(200)
	return nil
}

func (lb *LoadBalancer) assignServer(newServer *shared.NewServer) {
	lb.Lock()
	defer lb.Unlock()
	for _, server := range lb.Servers {
		if server.URL == newServer.URL {
			server.Status = ServerStatusHEALTHY
			return
		}
	}
	lb.Servers = append(lb.Servers, &Server{
		URL:    newServer.URL,
		Status: ServerStatusHEALTHY,
	})
}

func (lb *LoadBalancer) healthCheck() {
	// Ticker that check every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			for _, server := range lb.Servers {
				go doHealthCheck(server)
			}
		}
	}
}

func doHealthCheck(server *Server) {
	resp, err := http.Get(server.URL + "/healthcheck")
	if err != nil || resp.StatusCode != http.StatusOK {
		server.handleHealthCheckError()
		return
	}
	server.handleHealthCheckSuccess()
}

func (lb *LoadBalancer) Stats(ctx echo.Context) error {
	res := map[string]interface{}{}
	res["total_hit_count"] = lb.TotalHit
	serverStats := map[string]interface{}{}
	for _, server := range lb.Servers {
		serverStats[server.URL] = map[string]interface{}{
			"status":    server.Status,
			"hit_count": server.HitCount,
		}
	}
	res["servers"] = serverStats
	ctx.JSON(200, res)
	return nil
}
