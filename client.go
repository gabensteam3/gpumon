package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
)

type GPUReport struct {
	Index         int     `json:"index"`
	Name          string  `json:"name"`
	TemperatureC  int     `json:"temperature_c"`
	ProcessCount  int     `json:"process_count"`
}

type HostReport struct {
	Hostname        string  `json:"hostname"`
	CPUUsagePercent float64 `json:"cpu_usage_percent"`
	MemoryUsedMB    int     `json:"memory_used_mb"`
	MemoryTotalMB   int     `json:"memory_total_mb"`
	DiskUsed        string  `json:"disk_used"`  // e.g., "40G"
	DiskTotal       string  `json:"disk_total"` // e.g., "100G"
}

func fetchJSON(url string, target interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return json.Unmarshal(body, target)
}

func parseGB(s string) float64 {
	// Very basic: assumes "XXG" format, doesn't handle TiB/MiB etc.
	val, err := strconv.ParseFloat(s[:len(s)-1], 64)
	if err != nil {
		return 0
	}
	return val
}

func checkGPUHealth(gpus []GPUReport) bool {
	allHealthy := true
	for _, gpu := range gpus {
		if gpu.ProcessCount < 1 {
			fmt.Printf("[GPU] ❌ GPU %s has no active processes\n", gpu.Name)
			allHealthy = false
		}
		if gpu.TemperatureC >= 90 {
			fmt.Printf("[GPU] 🔥 GPU %s is too hot (%d°C)\n", gpu.Name, gpu.TemperatureC)
			allHealthy = false
		}
	}
	if allHealthy {
		fmt.Println("[GPU] ✅ All GPUs are healthy")
	}
	return allHealthy
}

func checkHostHealth(hosts []HostReport) bool {
	allHealthy := true
	for _, host := range hosts {
		memPercent := float64(host.MemoryUsedMB) / float64(host.MemoryTotalMB) * 100
		diskUsed := parseGB(host.DiskUsed)
		diskTotal := parseGB(host.DiskTotal)
		diskPercent := (diskUsed / diskTotal) * 100

		if host.CPUUsagePercent >= 90 {
			fmt.Printf("[Host %s] ⚠️ CPU usage too high: %.1f%%\n", host.Hostname, host.CPUUsagePercent)
			allHealthy = false
		}
		if memPercent >= 90 {
			fmt.Printf("[Host %s] ⚠️ Memory usage too high: %.1f%%\n", host.Hostname, memPercent)
			allHealthy = false
		}
		if diskPercent >= 90 {
			fmt.Printf("[Host %s] ⚠️ Disk usage too high: %.1f%%\n", host.Hostname, diskPercent)
			allHealthy = false
		}
	}
	if allHealthy {
		fmt.Println("[Host] ✅ All hosts are healthy")
	}
	return allHealthy
}

func main() {
	gpus := []GPUReport{}
	err := fetchJSON("http://192.168.0.1:1101/gpu/list", &gpus)
	if err != nil {
		fmt.Println("❌ System has issues")
		log.Fatal("Failed to fetch GPU list:", err)
	}

	hosts := []HostReport{}
	err = fetchJSON("http://192.168.0.1:1101/host/list", &hosts)
	if err != nil {
		fmt.Println("❌ System has issues")
		log.Fatal("Failed to fetch host metrics:", err)
	}

	gpuOk := checkGPUHealth(gpus)
	hostOk := checkHostHealth(hosts)

	if gpuOk && hostOk {
		fmt.Println("✅ System is healthy")
	} else {
		fmt.Println("❌ System has issues")
	}
}


