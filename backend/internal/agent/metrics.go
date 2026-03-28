package agent

import (
	"bufio"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

type RuntimeMetrics struct {
	CPUCount       int32
	CPUUsagePct    float64
	MemoryTotal    int64
	MemoryUsed     int64
	MemoryUsagePct float64
	DiskTotal      int64
	DiskUsed       int64
	DiskUsagePct   float64
}

// MetricsCollector 通过本机系统文件采集资源指标，不依赖额外三方库。
type MetricsCollector struct {
	mu        sync.Mutex
	hasCPURef bool
	prevTotal uint64
	prevIdle  uint64
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{}
}

func (c *MetricsCollector) Collect() RuntimeMetrics {
	m := RuntimeMetrics{
		CPUCount: int32(runtime.NumCPU()),
	}

	m.CPUUsagePct = c.readCPUUsage()
	m.MemoryTotal, m.MemoryUsed = readMemoryUsage()
	if m.MemoryTotal > 0 {
		m.MemoryUsagePct = percent(m.MemoryUsed, m.MemoryTotal)
	}

	m.DiskTotal, m.DiskUsed = readDiskUsage("/")
	if m.DiskTotal > 0 {
		m.DiskUsagePct = percent(m.DiskUsed, m.DiskTotal)
	}

	return m
}

func (c *MetricsCollector) readCPUUsage() float64 {
	total, idle, ok := readCPUStat()
	if !ok {
		return 0
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 首次采样没有基线，返回 0，下一次开始计算 delta 百分比。
	if !c.hasCPURef {
		c.prevTotal = total
		c.prevIdle = idle
		c.hasCPURef = true
		return 0
	}

	totalDelta := total - c.prevTotal
	idleDelta := idle - c.prevIdle
	c.prevTotal = total
	c.prevIdle = idle

	if totalDelta == 0 || totalDelta < idleDelta {
		return 0
	}

	usedDelta := totalDelta - idleDelta
	usage := float64(usedDelta) * 100 / float64(totalDelta)
	return clampPercent(usage)
}

func readCPUStat() (total uint64, idle uint64, ok bool) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0, false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return 0, 0, false
	}

	fields := strings.Fields(scanner.Text())
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0, 0, false
	}

	var values []uint64
	for _, f := range fields[1:] {
		v, err := strconv.ParseUint(f, 10, 64)
		if err != nil {
			return 0, 0, false
		}
		values = append(values, v)
		total += v
	}

	// idle + iowait 作为空闲时间。
	idle = values[3]
	if len(values) > 4 {
		idle += values[4]
	}
	return total, idle, true
}

func readMemoryUsage() (total int64, used int64) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	defer file.Close()

	var memTotalKB int64
	var memAvailableKB int64

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "MemTotal:"):
			memTotalKB = parseMeminfoLineKB(line)
		case strings.HasPrefix(line, "MemAvailable:"):
			memAvailableKB = parseMeminfoLineKB(line)
		}
		if memTotalKB > 0 && memAvailableKB > 0 {
			break
		}
	}

	if memTotalKB <= 0 {
		return 0, 0
	}
	if memAvailableKB < 0 {
		memAvailableKB = 0
	}

	total = memTotalKB * 1024
	used = total - (memAvailableKB * 1024)
	if used < 0 {
		used = 0
	}
	return total, used
}

func parseMeminfoLineKB(line string) int64 {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0
	}
	v, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return 0
	}
	return v
}

func readDiskUsage(path string) (total int64, used int64) {
	var fs syscall.Statfs_t
	if err := syscall.Statfs(path, &fs); err != nil {
		return 0, 0
	}

	blockSize := int64(fs.Bsize)
	if blockSize <= 0 {
		return 0, 0
	}

	total = int64(fs.Blocks) * blockSize
	free := int64(fs.Bavail) * blockSize
	used = total - free
	if used < 0 {
		used = 0
	}
	return total, used
}

func percent(used, total int64) float64 {
	if total <= 0 || used <= 0 {
		return 0
	}
	return clampPercent(float64(used) * 100 / float64(total))
}

func clampPercent(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return math.Round(v*100) / 100
}
