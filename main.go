package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"
	"io"
		"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type GPUReport struct {
	Index                  int    `json:"index"`
	Name                   string `json:"name"`
	FanPercent             int    `json:"fan_percent"`
	TemperatureC           int    `json:"temperature_c"`
	PowerWatt              float64 `json:"power_watt"`           // <-- updated
	MemoryUsedMiB          int    `json:"memory_used_mib"`
	MemoryTotalMiB         int    `json:"memory_total_mib"`
	UtilizationGpuPercent  int    `json:"utilization_gpu_percent"`
	ProcessCount           int    `json:"process_count"`
	ProcessNames           string `json:"process_names"`
	UpdatedAt              string  `json:"updated_at"` // ISO string
}

type HostReport struct {
	Hostname         string  `json:"hostname"`
	CPUUsagePercent  float64 `json:"cpu_usage_percent"`
	MemoryUsedMB     int     `json:"memory_used_mb"`
	MemoryTotalMB    int     `json:"memory_total_mb"`
	DiskUsed         string  `json:"disk_used"`
	DiskTotal        string  `json:"disk_total"`
	UpdatedAt        string  `json:"updated_at"`
}

type HardwareReport struct {
	Hostname string          `json:"hostname"`
	Uptime   string          `json:"uptime"`
	Kernel   string          `json:"kernel"`
	Distro   string          `json:"distro"`
	CPU      string          `json:"cpu"`
	Memory   string          `json:"memory"`
	Disk     json.RawMessage `json:"disk"`
	PCI      string          `json:"pci"`
	USB      string          `json:"usb"`
	Network  json.RawMessage `json:"network"`
	Storage  string          `json:"storage"`
}


func main() {
	db, err := sql.Open("sqlite3", "./gpu_inventory.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

_, err = db.Exec(`CREATE TABLE IF NOT EXISTS hardware_reports (
	hostname TEXT PRIMARY KEY,
	uptime TEXT,
	kernel TEXT,
	distro TEXT,
	cpu TEXT,
	memory TEXT,
	disk_json TEXT,
	pci TEXT,
	usb TEXT,
	network_json TEXT,
	storage TEXT,
	updated_at DATETIME
)`)
if err != nil {
	log.Fatal(err)
}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS host_metrics (
	hostname TEXT PRIMARY KEY,
	cpu_usage_percent REAL,
	memory_used_mb INTEGER,
	memory_total_mb INTEGER,
	disk_used TEXT,
	disk_total TEXT,
	updated_at DATETIME
)`)
if err != nil {
	log.Fatal(err)
}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS gpu_inventory (
		index_id INTEGER,
		name TEXT,
		fan_percent INTEGER,
		temperature_c INTEGER,
		power_watt INTEGER,
		memory_used_mib INTEGER,
		memory_total_mib INTEGER,
		utilization_gpu_percent INTEGER,
		process_count INTEGER,
		process_names TEXT,
		updated_at DATETIME,
		PRIMARY KEY(name)
	)`)
	if err != nil {
		log.Fatal(err)
	}

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)
	http.HandleFunc("/gpu/list", func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(`SELECT index_id, name, fan_percent, temperature_c, power_watt,
		memory_used_mib, memory_total_mib, utilization_gpu_percent,
		process_count, process_names, updated_at FROM gpu_inventory`)
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var gpus []GPUReport
		for rows.Next() {
			var gpu GPUReport
			var updatedAt time.Time
			err := rows.Scan(&gpu.Index, &gpu.Name, &gpu.FanPercent, &gpu.TemperatureC,
			&gpu.PowerWatt, &gpu.MemoryUsedMiB, &gpu.MemoryTotalMiB,
			&gpu.UtilizationGpuPercent, &gpu.ProcessCount, &gpu.ProcessNames, &updatedAt)
			gpu.UpdatedAt = updatedAt.Format(time.RFC3339)
			if err != nil {
				http.Error(w, "Scan error", http.StatusInternalServerError)
				return
			}
			gpus = append(gpus, gpu)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gpus)
	})

http.HandleFunc("/hardware/report", func(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}


	var report HardwareReport
	if err := json.Unmarshal(body, &report); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Optional: Print parsed report
	log.Printf("Received hardware report from host: %s", report.Hostname)

	stmt, err := db.Prepare(`INSERT INTO hardware_reports
	(hostname, uptime, kernel, distro, cpu, memory, disk_json, pci, usb, network_json, storage, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(hostname) DO UPDATE SET
		uptime=excluded.uptime,
		kernel=excluded.kernel,
		distro=excluded.distro,
		cpu=excluded.cpu,
		memory=excluded.memory,
		disk_json=excluded.disk_json,
		pci=excluded.pci,
		usb=excluded.usb,
		network_json=excluded.network_json,
		storage=excluded.storage,
		updated_at=excluded.updated_at`)
	if err != nil {
		http.Error(w, "DB prepare error", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		report.Hostname,
		report.Uptime,
		report.Kernel,
		report.Distro,
		report.CPU,
		report.Memory,
		string(report.Disk),
		report.PCI,
		report.USB,
		string(report.Network),
		report.Storage,
		time.Now(),
	)
	if err != nil {
		http.Error(w, "DB insert error", http.StatusInternalServerError)
		return
	}

	// Optional: save to DB or just acknowledge
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hardware report received"))
})

http.HandleFunc("/hardware/list", func(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT hostname, uptime, kernel, distro, cpu, memory, disk_json, pci, usb, network_json, storage, updated_at FROM hardware_reports`)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var reports []HardwareReport
	for rows.Next() {
		var hr HardwareReport
		var updatedAt time.Time
		var diskJSON, networkJSON string

		err := rows.Scan(&hr.Hostname, &hr.Uptime, &hr.Kernel, &hr.Distro, &hr.CPU, &hr.Memory, &diskJSON, &hr.PCI, &hr.USB, &networkJSON, &hr.Storage, &updatedAt)
		if err != nil {
			http.Error(w, "Scan error", http.StatusInternalServerError)
			return
		}

		hr.Disk = json.RawMessage(diskJSON)
		hr.Network = json.RawMessage(networkJSON)
		// Optional: you can add hr.UpdatedAt if needed

		reports = append(reports, hr)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reports)
})

http.HandleFunc("/host/report", func(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}
	log.Println("Received host body:", string(body))

	var report HostReport
	if err := json.Unmarshal(body, &report); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare(`INSERT INTO host_metrics
		(hostname, cpu_usage_percent, memory_used_mb, memory_total_mb, disk_used, disk_total, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(hostname) DO UPDATE SET
			cpu_usage_percent=excluded.cpu_usage_percent,
			memory_used_mb=excluded.memory_used_mb,
			memory_total_mb=excluded.memory_total_mb,
			disk_used=excluded.disk_used,
			disk_total=excluded.disk_total,
			updated_at=excluded.updated_at`)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(report.Hostname, report.CPUUsagePercent, report.MemoryUsedMB, report.MemoryTotalMB, report.DiskUsed, report.DiskTotal, time.Now())
	if err != nil {
		http.Error(w, "Insert error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
})


http.HandleFunc("/host/list", func(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT hostname, cpu_usage_percent, memory_used_mb, memory_total_mb, disk_used, disk_total, updated_at FROM host_metrics`)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var hosts []HostReport
	for rows.Next() {
		var h HostReport
		var updatedAt time.Time
		err := rows.Scan(&h.Hostname, &h.CPUUsagePercent, &h.MemoryUsedMB, &h.MemoryTotalMB, &h.DiskUsed, &h.DiskTotal, &updatedAt)
		h.UpdatedAt = updatedAt.Format(time.RFC3339)
		if err != nil {
			http.Error(w, "Scan error", http.StatusInternalServerError)
			return
		}
		hosts = append(hosts, h)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(hosts)
})


	http.HandleFunc("/gpu/report", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusInternalServerError)
			return
		}
		log.Println("Received body:", string(body))

		var gpus []GPUReport
		err = json.Unmarshal(body, &gpus)
		if err != nil {
			http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
			log.Println("JSON decode error:", err)
			return
		}

		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}

		stmt, err := tx.Prepare(`INSERT INTO gpu_inventory
		(index_id, name, fan_percent, temperature_c, power_watt, memory_used_mib,
		memory_total_mib, utilization_gpu_percent, process_count, process_names, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
		name=excluded.name,
		fan_percent=excluded.fan_percent,
		temperature_c=excluded.temperature_c,
		power_watt=excluded.power_watt,
		memory_used_mib=excluded.memory_used_mib,
		memory_total_mib=excluded.memory_total_mib,
		utilization_gpu_percent=excluded.utilization_gpu_percent,
		process_count=excluded.process_count,
		process_names=excluded.process_names,
		updated_at=excluded.updated_at;
		`)
		if err != nil {
			http.Error(w, "DB prepare error", http.StatusInternalServerError)
			return
		}
		defer stmt.Close()

		for _, gpu := range gpus {
			_, err := stmt.Exec(
				gpu.Index,
				gpu.Name,
				gpu.FanPercent,
				gpu.TemperatureC,
				gpu.PowerWatt,
				gpu.MemoryUsedMiB,
				gpu.MemoryTotalMiB,
				gpu.UtilizationGpuPercent,
				gpu.ProcessCount,
				gpu.ProcessNames,
				time.Now(),
			)
			if err != nil {
				tx.Rollback()
				http.Error(w, "DB insert error", http.StatusInternalServerError)
				return
			}
		}

		tx.Commit()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

http.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
	var issues []string
	cutoff := time.Now().Add(-5 * time.Minute)

	// Check GPUs
	rows, err := db.Query(`SELECT name, temperature_c, process_count, updated_at FROM gpu_inventory`)
	if err != nil {
		http.Error(w, "Failed to query GPUs", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var temp, count int
		var updatedAt time.Time

		if err := rows.Scan(&name, &temp, &count, &updatedAt); err != nil {
			http.Error(w, "Failed to scan GPU data", http.StatusInternalServerError)
			return
		}

		if updatedAt.Before(cutoff) {
			issues = append(issues, fmt.Sprintf("GPU %s data is stale (last update: %s)", name, updatedAt.Format(time.RFC3339)))
		}
		if count < 1 {
			issues = append(issues, fmt.Sprintf("GPU %s has no running processes", name))
		}
		if temp >= 90 {
			issues = append(issues, fmt.Sprintf("GPU %s temperature is high (%d°C)", name, temp))
		}
	}

	// Check Host Metrics
	hostRows, err := db.Query(`SELECT hostname, cpu_usage_percent, memory_used_mb, memory_total_mb, disk_used, disk_total, updated_at FROM host_metrics`)
	if err != nil {
		http.Error(w, "Failed to query host metrics", http.StatusInternalServerError)
		return
	}
	defer hostRows.Close()

	for hostRows.Next() {
		var hostname, diskUsedStr, diskTotalStr string
		var cpuPercent float64
		var memUsed, memTotal int
		var updatedAt time.Time

		if err := hostRows.Scan(&hostname, &cpuPercent, &memUsed, &memTotal, &diskUsedStr, &diskTotalStr, &updatedAt); err != nil {
			http.Error(w, "Failed to scan host data", http.StatusInternalServerError)
			return
		}

		if updatedAt.Before(cutoff) {
			issues = append(issues, fmt.Sprintf("Host %s data is stale (last update: %s)", hostname, updatedAt.Format(time.RFC3339)))
		}

		memUsage := float64(memUsed) / float64(memTotal) * 100

		var diskUsed, diskTotal int64
		fmt.Sscanf(diskUsedStr, "%d", &diskUsed)
		fmt.Sscanf(diskTotalStr, "%d", &diskTotal)
		var diskUsage float64
		if diskTotal > 0 {
			diskUsage = float64(diskUsed) / float64(diskTotal) * 100
		}

		if cpuPercent > 90 {
			issues = append(issues, fmt.Sprintf("Host %s CPU usage is high (%.1f%%)", hostname, cpuPercent))
		}
		if memUsage > 90 {
			issues = append(issues, fmt.Sprintf("Host %s memory usage is high (%.1f%%)", hostname, memUsage))
		}
		if diskUsage > 90 {
			issues = append(issues, fmt.Sprintf("Host %s disk usage is high (%.1f%%)", hostname, diskUsage))
		}
	}

	if len(issues) == 0 {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "unhealthy",
			"issues": issues,
		})
	}
})



	log.Println("Listening on :1101...")
	log.Fatal(http.ListenAndServe(":1101", nil))
}

