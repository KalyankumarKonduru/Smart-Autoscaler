package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/cors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type MetricsSnapshot struct {
	CPU float64   `json:"cpu"`    // millicores per pod avg
	Mem float64   `json:"mem"`    // MiB per pod avg
	P95 float64   `json:"p95"`    // ms
	At time.Time `json:"at"`
}

type Decision struct {
	Time      time.Time `json:"time"`
	Action    string    `json:"action"` // scale_up/down/hold
	From      int32     `json:"from"`
	To        int32     `json:"to"`
	Reason    string    `json:"reason"`
	Threshold string    `json:"threshold"`
}

type Config struct {
	Namespace      string
	Deployment     string
	MinReplicas    int32
	MaxReplicas    int32
	CPUThreshold   float64 // millicores
	MemThreshold   float64 // MiB
	P95Threshold   float64 // ms
	Cooldown       time.Duration
}

var (
	cfg Config
	kube *kubernetes.Clientset
	lastScale time.Time
	mu sync.Mutex
	currentReplicas int32
	events []Decision
)

func loadConfig() Config {
	min, _ := strconv.Atoi(getEnv("MIN_REPLICAS", "1"))
	max, _ := strconv.Atoi(getEnv("MAX_REPLICAS", "10"))
	cpu, _ := strconv.ParseFloat(getEnv("CPU_THRESHOLD", "500"), 64)
	mem, _ := strconv.ParseFloat(getEnv("MEM_THRESHOLD", "512"), 64)
	p95, _ := strconv.ParseFloat(getEnv("P95_THRESHOLD", "400"), 64)
	cool, _ := time.ParseDuration(getEnv("COOLDOWN", "60s"))
	return Config{
		Namespace:    getEnv("TARGET_NAMESPACE", "smart-autoscaler"),
		Deployment:   getEnv("TARGET_DEPLOYMENT", "sample-app"),
		MinReplicas:  int32(min),
		MaxReplicas:  int32(max),
		CPUThreshold: cpu,
		MemThreshold: mem,
		P95Threshold: p95,
		Cooldown:     cool,
	}
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" { return v }
	return d
}

func fetchMetrics() MetricsSnapshot {
	// Placeholder: In production, query Prometheus HTTP API for CPU/mem/p95.
	// For demo, read optional env overrides (useful in tests/k6 predict).
	cpu, _ := strconv.ParseFloat(getEnv("FAKE_CPU", "250"), 64)
	mem, _ := strconv.ParseFloat(getEnv("FAKE_MEM", "256"), 64)
	p95, _ := strconv.ParseFloat(getEnv("FAKE_P95", "120"), 64)
	return MetricsSnapshot{CPU: cpu, Mem: mem, P95: p95, At: time.Now()}
}

func decide(replica int32, m MetricsSnapshot, c Config) (int32, string) {
	// Simple rule: if any metric > threshold -> scale up by 1; if all well below -> scale down by 1.
	up := m.CPU > c.CPUThreshold || m.Mem > c.MemThreshold || m.P95 > c.P95Threshold
	down := m.CPU < c.CPUThreshold*0.5 && m.Mem < c.MemThreshold*0.5 && m.P95 < c.P95Threshold*0.5

	if up && replica < c.MaxReplicas { return replica+1, "Any(metric)>threshold" }
	if down && replica > c.MinReplicas { return replica-1, "All(metric)<half-threshold" }
	return replica, "Hold"
}

func scaleTo(ctx context.Context, n int32) error {
	dep, err := kube.AppsV1().Deployments(cfg.Namespace).Get(ctx, cfg.Deployment, metav1.GetOptions{})
	if err != nil { return err }
	*dep.Spec.Replicas = n
	_, err = kube.AppsV1().Deployments(cfg.Namespace).Update(ctx, dep, metav1.UpdateOptions{})
	return err
}

func reconcile(ctx context.Context) {
	mu.Lock()
	defer mu.Unlock()

	if time.Since(lastScale) < cfg.Cooldown { return }
	m := fetchMetrics()
	next, why := decide(currentReplicas, m, cfg)
	act := "hold"
	if next != currentReplicas {
		if err := scaleTo(ctx, next); err != nil {
			log.Printf("scale error: %v", err)
			return
		}
		act = "scale"
		lastScale = time.Now()
		currentReplicas = next
	}
	events = append(events, Decision{
		Time: time.Now(), Action: act, From: currentReplicas, To: next,
		Reason: why, Threshold: thresholdsString(),
	})
	if len(events) > 500 { events = events[len(events)-500:] }
}

func thresholdsString() string {
	return "CPU>"+strconv.FormatFloat(cfg.CPUThreshold,'f',0,64)+
		",Mem>"+strconv.FormatFloat(cfg.MemThreshold,'f',0,64)+
		",P95>"+strconv.FormatFloat(cfg.P95Threshold,'f',0,64)
}

func main() {
	cfg = loadConfig()
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("kube config: %v", err)
	}
	kube, err = kubernetes.NewForConfig(config)
	if err != nil { log.Fatalf("client: %v", err) }

	dep, err := kube.AppsV1().Deployments(cfg.Namespace).Get(context.Background(), cfg.Deployment, metav1.GetOptions{})
	if err == nil && dep.Spec.Replicas != nil { currentReplicas = *dep.Spec.Replicas } else { currentReplicas = cfg.MinReplicas }

	// HTTP API: /events, /predict (what-if), /healthz
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"}, // or restrict to your frontend domain later
  		AllowMethods: []string{"GET","POST","OPTIONS"},
  		AllowHeaders: []string{"Origin","Content-Type"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))
	r.GET("/healthz", func(c *gin.Context){ c.String(200, "ok") })
	r.GET("/events", func(c *gin.Context){ c.JSON(200, events) })
	r.POST("/predict", func(c *gin.Context){
		// Input: JSON { rps: number } -> converts to synthetic metrics, runs decide without scaling
		var body struct{ RPS float64 `json:"rps"` }
		if err := c.BindJSON(&body); err != nil { c.String(400, "bad json"); return }
		m := synthesizeMetrics(body.RPS)
		next, why := decide(currentReplicas, m, cfg)
		c.JSON(200, gin.H{"from": currentReplicas, "to": next, "reason": why, "metrics": m})
	})

	go func(){
		for {
			reconcile(context.Background())
			time.Sleep(5 * time.Second)
		}
	}()

	log.Println("autoscaler running")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}

func synthesizeMetrics(rps float64) MetricsSnapshot {
	// naive mapping: CPU ~ 2*rps, Mem ~ 1*rps, P95 grows when rps > 200
	cpu := 2*rps
	mem := 1*rps
	p95 := 100.0 + max(0, rps-200)*0.8
	return MetricsSnapshot{CPU: cpu, Mem: mem, P95: p95, At: time.Now()}
}

func max(a, b float64) float64 { if a>b { return a }; return b }
