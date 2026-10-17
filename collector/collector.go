package collector

import (
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

type CPU struct {
	Model        string  `json:"model"`
	Cores        int     `json:"cores"`
	UsagePercent float64 `json:"usage_percent"`
	Load1        float64 `json:"load1"`
	Load5        float64 `json:"load5"`
	Load15       float64 `json:"load15"`
}

type Mem struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Available   uint64  `json:"available"`
	UsedPercent float64 `json:"used_percent"`
}

type Swap struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"used_percent"`
}

type Temp struct {
	Label   string  `json:"label"`
	Celsius float64 `json:"celsius"`
}

type Disk struct {
	Mount       string  `json:"mount"`
	Fs          string  `json:"fs"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Available   uint64  `json:"available"`
	UsedPercent float64 `json:"used_percent"`
}

type Metrics struct {
	Hostname  string  `json:"hostname"`
	OS        string  `json:"os"`
	Arch      string  `json:"arch"`
	UptimeSec float64 `json:"uptime_s"`
	Timestamp string  `json:"timestamp"`
	CPU       CPU     `json:"cpu"`
	Mem       Mem     `json:"mem"`
	Swap      Swap    `json:"swap"`
	Temps     []Temp  `json:"temps"`
	Disks     []Disk  `json:"disks"`
}

type Collector struct {
	mu       sync.RWMutex
	hostname string
	cpuPct   float64
	cpuModel string
	cores    int
	disks    []string
	stop     chan struct{}
	started  bool
}

func New() *Collector {
	host, _ := os.Hostname()
	if host == "" {
		host = "unknown"
	}
	c := &Collector{
		cpuModel: cpuModel(),
		cores:    cpuCount(),
		disks:    diskMounts(),
		stop:     make(chan struct{}),
	}
	c.hostname = host
	return c
}

func (c *Collector) Start(interval time.Duration) {
	if c.started {
		return
	}
	if interval <= 0 {
		interval = time.Second
	}
	c.started = true
	go c.sampleLoop(interval)
}

func (c *Collector) Stop() {
	if c.started {
		close(c.stop)
	}
}

func (c *Collector) sampleLoop(interval time.Duration) {
	prev := readCPUStat()
	t := time.NewTicker(interval)
	defer t.Stop()
	c.setCPU(usage(prev, prev))
	for {
		select {
		case <-c.stop:
			return
		case <-t.C:
			cur := readCPUStat()
			c.setCPU(usage(prev, cur))
			prev = cur
		}
	}
}

func (c *Collector) setCPU(p float64) {
	c.mu.Lock()
	c.cpuPct = p
	c.mu.Unlock()
}

func (c *Collector) Collect() Metrics {
	c.mu.RLock()
	pct := c.cpuPct
	c.mu.RUnlock()

	if pct == 0 {
		prev := readCPUStat()
		time.Sleep(150 * time.Millisecond)
		cur := readCPUStat()
		pct = usage(prev, cur)
	}

	l1, l5, l15 := loadavg()
	mem, swap := memInfo()
	temps := temps()
	sort.Slice(temps, func(i, j int) bool { return temps[i].Label < temps[j].Label })

	return Metrics{
		Hostname:  c.hostname,
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		UptimeSec: uptime(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		CPU: CPU{
			Model:        c.cpuModel,
			Cores:        c.cores,
			UsagePercent: round1(pct),
			Load1:        l1,
			Load5:        l5,
			Load15:       l15,
		},
		Mem:   mem,
		Swap:  swap,
		Temps: temps,
		Disks: disks(c.disks),
	}
}

func normModel(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}
