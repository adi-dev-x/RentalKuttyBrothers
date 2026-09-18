package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// Config holds runtime configuration loaded from environment variables.
type Config struct {
	TableName     string        // DynamoDB table storing the activity timestamp
	ActivityKey   string        // Partition key value identifying the activity record
	InstanceID    string        // EC2 instance ID to stop / proxy to
	IdleThreshold time.Duration // How long the instance must be idle before stopping
	Region        string        // AWS region
	BackendPort   int           // Port on the EC2 instance to proxy to (e.g. 8080)
	BackendScheme string        // "http" or "https"
	ProxyTimeout  time.Duration // Per-request timeout for the upstream call
	WakeOnRequest bool          // If true, start a stopped instance on incoming request
}

// ActivityStore abstracts the backend that holds the last-activity timestamp.
type ActivityStore interface {
	GetLastActivity(ctx context.Context) (time.Time, error)
	PutLastActivity(ctx context.Context, t time.Time) error
}

// EC2Controller covers everything we need to do against EC2.
type EC2Controller interface {
	StopInstance(ctx context.Context, instanceID string) error
	StartInstance(ctx context.Context, instanceID string) error
	GetInstance(ctx context.Context, instanceID string) (state ec2types.InstanceStateName, privateIP, publicIP string, err error)
}

// ShutdownResponse is what the scheduled (shutdown) path returns.
type ShutdownResponse struct {
	Stopped      bool      `json:"stopped"`
	Reason       string    `json:"reason"`
	LastActivity time.Time `json:"lastActivity"`
	IdleFor      string    `json:"idleFor"`
	InstanceID   string    `json:"instanceId"`
}

// --- DynamoDB implementation of ActivityStore ---

type dynamoStore struct {
	client    *dynamodb.Client
	tableName string
	key       string
}

func (d *dynamoStore) GetLastActivity(ctx context.Context) (time.Time, error) {
	out, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: d.key},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return time.Time{}, fmt.Errorf("dynamodb get item: %w", err)
	}
	if out.Item == nil {
		return time.Time{}, nil
	}
	tsAttr, ok := out.Item["lastActivity"]
	if !ok {
		return time.Time{}, fmt.Errorf("lastActivity attribute missing on record %q", d.key)
	}
	tsStr, ok := tsAttr.(*types.AttributeValueMemberS)
	if !ok {
		return time.Time{}, fmt.Errorf("lastActivity has unexpected type")
	}
	t, err := time.Parse(time.RFC3339, tsStr.Value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse lastActivity: %w", err)
	}
	return t, nil
}

func (d *dynamoStore) PutLastActivity(ctx context.Context, t time.Time) error {
	_, err := d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.tableName),
		Item: map[string]types.AttributeValue{
			"id":           &types.AttributeValueMemberS{Value: d.key},
			"lastActivity": &types.AttributeValueMemberS{Value: t.UTC().Format(time.RFC3339)},
		},
	})
	if err != nil {
		return fmt.Errorf("dynamodb put item: %w", err)
	}
	return nil
}

// --- EC2 implementation of EC2Controller ---

type ec2Controller struct {
	client *ec2.Client
}

func (e *ec2Controller) GetInstance(ctx context.Context, instanceID string) (ec2types.InstanceStateName, string, string, error) {
	desc, err := e.client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return "", "", "", fmt.Errorf("describe instance: %w", err)
	}
	if len(desc.Reservations) == 0 || len(desc.Reservations[0].Instances) == 0 {
		return "", "", "", fmt.Errorf("instance %s not found", instanceID)
	}
	inst := desc.Reservations[0].Instances[0]
	priv, pub := "", ""
	if inst.PrivateIpAddress != nil {
		priv = *inst.PrivateIpAddress
	}
	if inst.PublicIpAddress != nil {
		pub = *inst.PublicIpAddress
	}
	return inst.State.Name, priv, pub, nil
}

func (e *ec2Controller) StopInstance(ctx context.Context, instanceID string) error {
	state, _, _, err := e.GetInstance(ctx, instanceID)
	if err != nil {
		return err
	}
	if state != ec2types.InstanceStateNameRunning {
		log.Printf("instance %s is in state %q; skipping stop", instanceID, state)
		return nil
	}
	_, err = e.client.StopInstances(ctx, &ec2.StopInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("stop instance: %w", err)
	}
	return nil
}

func (e *ec2Controller) StartInstance(ctx context.Context, instanceID string) error {
	state, _, _, err := e.GetInstance(ctx, instanceID)
	if err != nil {
		return err
	}
	if state == ec2types.InstanceStateNameRunning || state == ec2types.InstanceStateNamePending {
		return nil
	}
	_, err = e.client.StartInstances(ctx, &ec2.StartInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("start instance: %w", err)
	}
	return nil
}

// --- Pure decision function (unit-testable, no AWS deps) ---

func decide(now, lastActivity time.Time, idleThreshold time.Duration) (bool, string) {
	if lastActivity.IsZero() {
		return false, "no activity record found; skipping stop"
	}
	idleFor := now.Sub(lastActivity)
	if idleFor < idleThreshold {
		return false, fmt.Sprintf("recent activity %s ago (threshold %s)", idleFor.Round(time.Second), idleThreshold)
	}
	return true, fmt.Sprintf("idle for %s (threshold %s)", idleFor.Round(time.Second), idleThreshold)
}

// --- IP cache so we don't DescribeInstances on every request ---

type ipCache struct {
	mu        sync.RWMutex
	ip        string
	expiresAt time.Time
	ttl       time.Duration
}

func (c *ipCache) get() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if time.Now().Before(c.expiresAt) {
		return c.ip
	}
	return ""
}

func (c *ipCache) set(ip string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ip = ip
	c.expiresAt = time.Now().Add(c.ttl)
}

func (c *ipCache) invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.expiresAt = time.Time{}
}

// --- Handler ---

type handler struct {
	cfg        Config
	store      ActivityStore
	ec2        EC2Controller
	now        func() time.Time
	httpClient *http.Client
	ips        *ipCache
}

// resolveBackendIP returns the IP we should proxy to, refreshing from EC2 if
// the cache is empty or stale.
func (h *handler) resolveBackendIP(ctx context.Context) (string, ec2types.InstanceStateName, error) {
	if ip := h.ips.get(); ip != "" {
		return ip, ec2types.InstanceStateNameRunning, nil
	}
	state, priv, pub, err := h.ec2.GetInstance(ctx, h.cfg.InstanceID)
	if err != nil {
		return "", "", err
	}
	// Prefer private IP (Lambda in same VPC); fall back to public.
	ip := priv
	if ip == "" {
		ip = pub
	}
	if ip != "" && state == ec2types.InstanceStateNameRunning {
		h.ips.set(ip)
	}
	return ip, state, nil
}

// --- Scheduled (shutdown) path ---

func (h *handler) handleShutdown(ctx context.Context) (ShutdownResponse, error) {
	last, err := h.store.GetLastActivity(ctx)
	if err != nil {
		return ShutdownResponse{}, fmt.Errorf("read last activity: %w", err)
	}

	now := h.now()
	shouldStop, reason := decide(now, last, h.cfg.IdleThreshold)

	resp := ShutdownResponse{
		LastActivity: last,
		IdleFor:      now.Sub(last).Round(time.Second).String(),
		InstanceID:   h.cfg.InstanceID,
		Reason:       reason,
	}

	if !shouldStop {
		log.Printf("not stopping instance: %s", reason)
		return resp, nil
	}

	log.Printf("stopping instance %s: %s", h.cfg.InstanceID, reason)
	if err := h.ec2.StopInstance(ctx, h.cfg.InstanceID); err != nil {
		return resp, fmt.Errorf("stop instance: %w", err)
	}
	h.ips.invalidate() // IP may change on next start
	resp.Stopped = true
	return resp, nil
}

// --- Proxy (frontend) path ---

func (h *handler) handleProxy(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	// 1. Update last activity timestamp first — even if the upstream call
	//    fails, we want to record that someone tried, so the shutdown sweep
	//    doesn't kill an instance that's mid-warmup.
	if err := h.store.PutLastActivity(ctx, h.now()); err != nil {
		log.Printf("warn: failed to update last activity: %v", err)
		// Non-fatal: proceed with proxying.
	}

	// 2. Find the backend IP / state.
	ip, state, err := h.resolveBackendIP(ctx)
	if err != nil {
		return jsonResponse(http.StatusBadGateway, map[string]string{
			"error": "failed to resolve backend: " + err.Error(),
		}), nil
	}

	// 3. If the instance isn't running, optionally start it and return 503.
	if state != ec2types.InstanceStateNameRunning {
		if h.cfg.WakeOnRequest && state == ec2types.InstanceStateNameStopped {
			if err := h.ec2.StartInstance(ctx, h.cfg.InstanceID); err != nil {
				log.Printf("warn: failed to start instance: %v", err)
			}
		}
		return jsonResponse(http.StatusServiceUnavailable, map[string]string{
			"error":  "backend is " + string(state),
			"status": string(state),
			"hint":   "retry in ~60s; instance is starting",
		}), nil
	}

	// 4. Build and forward the request.
	upstreamResp, err := h.forward(ctx, req, ip)
	if err != nil {
		// On connection failures, the cached IP is probably stale (instance
		// restarted, ENI changed). Invalidate and let the next call refresh.
		h.ips.invalidate()
		return jsonResponse(http.StatusBadGateway, map[string]string{
			"error": "upstream call failed: " + err.Error(),
		}), nil
	}
	return upstreamResp, nil
}

func (h *handler) forward(ctx context.Context, req events.LambdaFunctionURLRequest, ip string) (events.LambdaFunctionURLResponse, error) {
	// Reconstruct the path. Function URLs put it in RawPath / RequestContext.HTTP.Path.
	path := req.RawPath
	if path == "" {
		path = req.RequestContext.HTTP.Path
	}
	if path == "" {
		path = "/"
	}
	upstreamURL := fmt.Sprintf("%s://%s:%d%s", h.cfg.BackendScheme, ip, h.cfg.BackendPort, path)
	if req.RawQueryString != "" {
		upstreamURL += "?" + req.RawQueryString
	}

	// Body — Function URLs base64-encode binary bodies.
	var body io.Reader
	if req.Body != "" {
		if req.IsBase64Encoded {
			decoded, err := base64.StdEncoding.DecodeString(req.Body)
			if err != nil {
				return events.LambdaFunctionURLResponse{}, fmt.Errorf("decode body: %w", err)
			}
			body = strings.NewReader(string(decoded))
		} else {
			body = strings.NewReader(req.Body)
		}
	}

	method := req.RequestContext.HTTP.Method
	if method == "" {
		method = http.MethodGet
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, upstreamURL, body)
	if err != nil {
		return events.LambdaFunctionURLResponse{}, fmt.Errorf("build request: %w", err)
	}

	// Copy headers, stripping hop-by-hop ones and the inbound Host.
	for k, v := range req.Headers {
		if isHopByHop(k) || strings.EqualFold(k, "host") {
			continue
		}
		httpReq.Header.Set(k, v)
	}
	// Standard proxy hints for the backend.
	if sourceIP := req.RequestContext.HTTP.SourceIP; sourceIP != "" {
		httpReq.Header.Set("X-Forwarded-For", sourceIP)
	}
	httpReq.Header.Set("X-Forwarded-Proto", "https")

	resp, err := h.httpClient.Do(httpReq)
	if err != nil {
		return events.LambdaFunctionURLResponse{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return events.LambdaFunctionURLResponse{}, fmt.Errorf("read upstream body: %w", err)
	}

	out := events.LambdaFunctionURLResponse{
		StatusCode: resp.StatusCode,
		Headers:    map[string]string{},
	}
	for k, vals := range resp.Header {
		if isHopByHop(k) {
			continue
		}
		out.Headers[k] = strings.Join(vals, ",")
	}

	// Detect if the body is text-ish; otherwise base64-encode it.
	ct := resp.Header.Get("Content-Type")
	if isTextual(ct) {
		out.Body = string(respBody)
	} else {
		out.Body = base64.StdEncoding.EncodeToString(respBody)
		out.IsBase64Encoded = true
	}
	return out, nil
}

// --- Event routing ---

// invoke dispatches based on event shape. Function URL events have a
// requestContext.http block with a method; scheduled events do not.
func (h *handler) invoke(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
	var probe struct {
		RequestContext struct {
			HTTP struct {
				Method string `json:"method"`
			} `json:"http"`
		} `json:"requestContext"`
	}
	_ = json.Unmarshal(raw, &probe)

	if probe.RequestContext.HTTP.Method != "" {
		var req events.LambdaFunctionURLRequest
		if err := json.Unmarshal(raw, &req); err != nil {
			return nil, fmt.Errorf("decode function url request: %w", err)
		}
		resp, err := h.handleProxy(ctx, req)
		if err != nil {
			return nil, err
		}
		return json.Marshal(resp)
	}

	resp, err := h.handleShutdown(ctx)
	if err != nil {
		return nil, err
	}
	return json.Marshal(resp)
}

// --- Helpers ---

func jsonResponse(status int, payload any) events.LambdaFunctionURLResponse {
	b, _ := json.Marshal(payload)
	return events.LambdaFunctionURLResponse{
		StatusCode: status,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(b),
	}
}

func isHopByHop(h string) bool {
	switch strings.ToLower(h) {
	case "connection", "keep-alive", "proxy-authenticate", "proxy-authorization",
		"te", "trailers", "transfer-encoding", "upgrade", "content-length":
		return true
	}
	return false
}

func isTextual(contentType string) bool {
	ct := strings.ToLower(contentType)
	return strings.HasPrefix(ct, "text/") ||
		strings.Contains(ct, "json") ||
		strings.Contains(ct, "xml") ||
		strings.Contains(ct, "javascript") ||
		strings.Contains(ct, "x-www-form-urlencoded")
}

// --- Config loading ---

func loadConfig() (Config, error) {
	cfg := Config{
		TableName:     os.Getenv("ACTIVITY_TABLE"),
		ActivityKey:   getEnvDefault("ACTIVITY_KEY", "api-last-activity"),
		InstanceID:    os.Getenv("EC2_INSTANCE_ID"),
		Region:        getEnvDefault("AWS_REGION", "us-east-1"),
		BackendScheme: getEnvDefault("BACKEND_SCHEME", "http"),
	}

	mins, err := strconv.Atoi(getEnvDefault("IDLE_THRESHOLD_MINUTES", "10"))
	if err != nil {
		return cfg, fmt.Errorf("invalid IDLE_THRESHOLD_MINUTES: %w", err)
	}
	cfg.IdleThreshold = time.Duration(mins) * time.Minute

	port, err := strconv.Atoi(getEnvDefault("BACKEND_PORT", "8080"))
	if err != nil {
		return cfg, fmt.Errorf("invalid BACKEND_PORT: %w", err)
	}
	cfg.BackendPort = port

	timeoutSec, err := strconv.Atoi(getEnvDefault("PROXY_TIMEOUT_SECONDS", "25"))
	if err != nil {
		return cfg, fmt.Errorf("invalid PROXY_TIMEOUT_SECONDS: %w", err)
	}
	cfg.ProxyTimeout = time.Duration(timeoutSec) * time.Second

	cfg.WakeOnRequest = strings.EqualFold(getEnvDefault("WAKE_ON_REQUEST", "true"), "true")

	if cfg.TableName == "" {
		return cfg, fmt.Errorf("ACTIVITY_TABLE env var is required")
	}
	if cfg.InstanceID == "" {
		return cfg, fmt.Errorf("EC2_INSTANCE_ID env var is required")
	}
	return cfg, nil
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// --- Entrypoint ---

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	awsCfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(cfg.Region))
	if err != nil {
		log.Fatalf("aws config: %v", err)
	}

	h := &handler{
		cfg:   cfg,
		store: &dynamoStore{client: dynamodb.NewFromConfig(awsCfg), tableName: cfg.TableName, key: cfg.ActivityKey},
		ec2:   &ec2Controller{client: ec2.NewFromConfig(awsCfg)},
		now:   time.Now,
		httpClient: &http.Client{
			Timeout: cfg.ProxyTimeout,
		},
		ips: &ipCache{ttl: 5 * time.Minute},
	}

	lambda.Start(h.invoke)
}
