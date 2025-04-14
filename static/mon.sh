#!/bin/bash

# Get GPU and process info
gpu_info=$(nvidia-smi --query-gpu=index,name,fan.speed,temperature.gpu,power.draw,memory.used,memory.total,utilization.gpu --format=csv,noheader,nounits)
process_info=$(nvidia-smi)

# Declare arrays
declare -a gpu_json

# Initialize process data as associative arrays
declare -A gpu_proc_count
declare -A gpu_proc_names

# Parse process table
in_process_section=false
while IFS= read -r line; do
    if [[ "$line" =~ "Processes:" ]]; then
        in_process_section=true
        continue
    fi
    if $in_process_section && [[ "$line" =~ ^\| ]]; then
        # Skip headers
        if [[ "$line" =~ "GPU" && "$line" =~ "PID" ]]; then
            continue
        fi
        gpu_id=$(echo "$line" | awk '{print $2}')
        process_path=$(echo "$line" | awk '{for(i=6;i<=NF-2;++i) printf $i " "; print ""}' | xargs)
        process_short=$(basename "$process_path")

        if [[ "$gpu_id" =~ ^[0-9]+$ ]]; then
            ((gpu_proc_count[$gpu_id]++))
            if [[ -z "${gpu_proc_names[$gpu_id]}" ]]; then
                gpu_proc_names[$gpu_id]="$process_short"
            else
                gpu_proc_names[$gpu_id]+=", $process_short"
            fi
        fi
    fi
done <<< "$process_info"

# Build JSON entries per GPU
while IFS=',' read -r index name fan temp power mem_used mem_total util; do
    index=$(echo "$index" | xargs)
    name=$(echo "$name" | xargs)
    fan=$(echo "$fan" | xargs | tr -d '%')
    temp=$(echo "$temp" | xargs)
    power=$(echo "$power" | xargs)
    mem_used=$(echo "$mem_used" | xargs)
    mem_total=$(echo "$mem_total" | xargs)
    util=$(echo "$util" | xargs)

    proc_count=${gpu_proc_count[$index]:-0}
    proc_names=${gpu_proc_names[$index]:-""}

    # Escape quotes in process names
    proc_names=$(echo "$proc_names" | sed 's/"/\\"/g')

    gpu_json+=("{
        \"index\": $index,
	\"name\": \"$index@$(cat /etc/hostname)@${name}\",
        \"fan_percent\": $fan,
        \"temperature_c\": $temp,
        \"power_watt\": $power,
        \"memory_used_mib\": $mem_used,
        \"memory_total_mib\": $mem_total,
        \"utilization_gpu_percent\": $util,
        \"process_count\": $proc_count,
        \"process_names\": \"${proc_names}\"
    }")
done <<< "$gpu_info"

# Output full JSON
echo "["
(IFS=,; echo "${gpu_json[*]}")
echo "]"


# Add this at the bottom of /root/monn.sh
json_output=$(cat <<EOF
[
$(IFS=,; echo "${gpu_json[*]}")
]
EOF
)

json_data_file=$(mktemp)
/bin/cat > "$json_data_file" <<EOF
[
$(IFS=,; echo "${gpu_json[*]}")
]
EOF

# Show JSON to debug if needed
cat "$json_data_file"|jq

curl -s -H "Content-Type: application/json" --data-binary "@$json_data_file" http://192.168.0.1:1101/gpu/report
rm -f "$json_data_file"

# Get the 1-minute load average from the `uptime` command
load_avg=$(uptime | awk -F'load average: ' '{ print $2 }' | cut -d',' -f1)

# Get the number of CPU cores
cpu_count=$(nproc)

# Calculate CPU usage as load average divided by number of CPUs
cpu_usage=$(echo "$load_avg $cpu_count" | awk '{ printf "%.2f", ($1 / $2) * 100 }')


mem_info=$(free -m | awk '/Mem:/ {print $3, $2}')
disk_info=$(df -h / | awk 'NR==2 {print $3, $2}')

read mem_used mem_total <<< "$mem_info"
read disk_used disk_total <<< "$disk_info"

hostname=$(cat /etc/hostname)

json=$(cat <<EOF
{
  "hostname": "$hostname",
  "cpu_usage_percent": $cpu_usage,
  "memory_used_mb": $mem_used,
  "memory_total_mb": $mem_total,
  "disk_used": "$disk_used",
  "disk_total": "$disk_total"
}
EOF
)

tmp=$(mktemp)
echo "$json" > "$tmp"
cat "$tmp" | jq

curl -s -H "Content-Type: application/json" --data-binary "@$tmp" http://192.168.0.1:1101/host/report
rm -f "$tmp"

sleep 200
exit 0

hostname=$(hostname)
uptime=$(uptime -p)
kernel=$(uname -r)
distro=$(cat /etc/os-release)
cpu=$(lscpu)
memory=$(free -h)
disk=$(lsblk -o NAME,SIZE,TYPE,MOUNTPOINT -J)
pci=$(lspci -mm )
usb=$(lsusb )
network=$(ip -j addr)
storage=$(df -h --output=source,fstype,size,used,avail,pcent,target -x tmpfs -x devtmpfs )

# Construct JSON
json=$(jq -n \
  --arg hostname "$hostname" \
  --arg uptime "$uptime" \
  --arg kernel "$kernel" \
  --arg distro "$distro" \
  --arg cpu "$cpu" \
  --arg memory "$memory" \
  --argjson disk "$disk" \
  --arg pci "$pci" \
  --arg usb "$usb" \
  --argjson network "$network" \
  --arg storage "$storage" \
  '{
    hostname: $hostname,
    uptime: $uptime,
    kernel: $kernel,
    distro: $distro,
    cpu: $cpu,
    memory: $memory,
    disk: $disk,
    pci: $pci,
    usb: $usb,
    network: $network,
    storage: $storage
  }')


tmp=$(mktemp)
echo "$json" > "$tmp"
cat "$tmp" | jq

curl -s -H "Content-Type: application/json" --data-binary "@$tmp" http://192.168.0.1:1101/hardware/report
rm -f "$tmp"


sleep 200
