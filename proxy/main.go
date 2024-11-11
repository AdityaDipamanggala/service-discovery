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
	"github.com/shopspring/decimal"
)

func main() {
	// Initiate load balancer handler
	lb := NewLoadBalancer()

	// Initiate healthcheck ticker
	go lb.healthCheck()
	e := echo.New()

	// Registered routes
	e.PUT("/reset", lb.ResetStatsHandler)
	e.GET("/stats", lb.StatsHandler)
	e.POST("/register", lb.RegisterServerHandler)
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
)

type Server struct {
	// Constant field assigned only when init
	URL                       string
	RequestErrorThreshold     int
	HealthCheckErrorThreshold int
	SlowRequestThreshold      int

	// Field that can be mutated each request
	sync.Mutex
	HitCount              decimal.Decimal
	Weight                int
	HealthCheckErrorCount int
	RequestErrorCount     int
	SlowRequestCount      int
	Status                ServerStatus
	RecoverTime           time.Time
	AverageLatency        decimal.Decimal
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
	if s.RequestErrorCount >= s.RequestErrorThreshold {
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
	if s.HealthCheckErrorCount >= s.HealthCheckErrorThreshold {
		s.Status = ServerStatusDOWN
	}
}

// Load balancer need collection of servers
type LoadBalancer struct {
	sync.Mutex
	Client          http.Client
	Servers         []*Server
	Counter         int
	WeightCounter   int
	NormalWeight    int
	SlowWeight      int
	TotalHit        decimal.Decimal
	AverageLatency  decimal.Decimal
	ExpectedLatency decimal.Decimal
}

// Initiate LoadBalancer class
func NewLoadBalancer() *LoadBalancer {
	return &LoadBalancer{
		Servers:         []*Server{},
		WeightCounter:   2,
		NormalWeight:    2,
		SlowWeight:      1,
		ExpectedLatency: decimal.NewFromInt(100),
	}
}

// ProxyHandler handle client request and pass it to application instance
func (lb *LoadBalancer) ProxyHandler(ctx echo.Context) error {
	if len(lb.Servers) == 0 {
		return echo.NewHTTPError(http.StatusRequestTimeout, "No healthy server")
	}
	// Select the server
	server := lb.selectServer()

	// Construct request from selected server
	reqUrl := fmt.Sprintf("%s%s?%s", server.URL, ctx.Request().URL.Path, ctx.Request().URL.RawQuery)
	req, err := http.NewRequest(ctx.Request().Method, reqUrl, ctx.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct request")
	}
	req.Header = ctx.Request().Header

	// Call and track the request latency
	startTime := time.Now()
	resp, err := lb.Client.Do(req)
	duration := time.Since(startTime)
	lb.processLatency(duration, server)
	if err != nil || resp.StatusCode != 200 {
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
	var server *Server

	// Reselect server if it's down or unhealthy but still in recover period
	for server == nil || server.Weight < lb.WeightCounter || server.Status == ServerStatusDOWN || (server.Status == ServerStatusUNHEALTHY && time.Now().Before(server.RecoverTime)) {
		idx := lb.Counter % len(lb.Servers)
		if idx == 0 {
			lb.WeightCounter--
			if lb.WeightCounter < 1 {
				lb.WeightCounter = lb.NormalWeight
			}
		}
		server = lb.Servers[idx]
		lb.Counter++
	}
	server.HitCount = server.HitCount.Add(decimal.NewFromInt(1))
	lb.TotalHit = lb.TotalHit.Add(decimal.NewFromInt(1))

	return server
}

// RegisterServerHandler will be triggered from application instance when initiated and register the metadata to the loadbalancer
func (lb *LoadBalancer) RegisterServerHandler(ctx echo.Context) error {
	newServer := &shared.NewServer{}
	if err := ctx.Bind(newServer); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to bind request")
	}
	lb.assignServer(newServer)
	ctx.Response().WriteHeader(200)
	return nil
}

// assignServer mutate the LoadBalancer object by appending new server
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
		URL:                       newServer.URL,
		RequestErrorThreshold:     2,
		HealthCheckErrorThreshold: 2,
		SlowRequestThreshold:      2,
		Status:                    ServerStatusHEALTHY,
		Weight:                    lb.NormalWeight,
	})
}

// Run ticker to frequently check the healthcheck of registered servers
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

// Execute the health check
func doHealthCheck(server *Server) {
	resp, err := http.Get(server.URL + "/healthcheck")
	if err != nil || resp.StatusCode != http.StatusOK {
		server.handleHealthCheckError()
		return
	}
	server.handleHealthCheckSuccess()
}

// StatsHandler return the internal statistics of the server collection
func (lb *LoadBalancer) StatsHandler(ctx echo.Context) error {
	res := map[string]interface{}{}
	res["total_hit_count"] = lb.TotalHit
	res["total_avg_latency"] = lb.AverageLatency
	serverStats := map[string]interface{}{}
	for _, server := range lb.Servers {
		waitTime := time.Until(server.RecoverTime).Seconds()
		serverStats[server.URL] = map[string]interface{}{
			"status":    server.Status,
			"hit_count": server.HitCount,
			"avg_lat":   server.AverageLatency,
			"weight":    server.Weight,
			"wait":      max(0, waitTime),
		}
	}
	res["servers"] = serverStats
	ctx.JSON(200, res)
	return nil
}

func (lb *LoadBalancer) ResetStatsHandler(ctx echo.Context) error {
	servers := lb.Servers
	lb.Servers = []*Server{}
	lb.WeightCounter = 2
	lb.TotalHit = decimal.Decimal{}
	lb.AverageLatency = decimal.Decimal{}
	for _, server := range servers {
		lb.Servers = append(lb.Servers, &Server{
			URL:                       server.URL,
			RequestErrorThreshold:     2,
			HealthCheckErrorThreshold: 2,
			SlowRequestThreshold:      2,
			Status:                    ServerStatusHEALTHY,
			Weight:                    lb.NormalWeight,
		})
	}
	return nil
}

// processLatency will mutate LoadBalancer.AverageLatency and server.AverageLatency and evaluate priority
func (lb *LoadBalancer) processLatency(latency time.Duration, server *Server) {
	lb.Lock()
	defer lb.Unlock()
	lb.AverageLatency = lb.AverageLatency.Mul(lb.TotalHit.Sub(decimal.NewFromInt(1))).Add(decimal.NewFromInt(latency.Milliseconds())).Div(lb.TotalHit)
	server.AverageLatency = server.AverageLatency.Mul(server.HitCount.Sub(decimal.NewFromInt(1))).Add(decimal.NewFromInt(latency.Milliseconds())).Div(server.HitCount)
	if decimal.NewFromInt(latency.Milliseconds()).LessThanOrEqual(lb.ExpectedLatency) {
		server.SlowRequestCount = 0
		server.Weight = lb.NormalWeight
		return
	}
	server.SlowRequestCount += 1
	if server.SlowRequestCount > server.SlowRequestThreshold {
		server.Weight = 1
	}
}
