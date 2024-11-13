package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	gomail "gopkg.in/mail.v2"
)

type SystemMetrics struct {
	Timestamp   time.Time
	CPUCount    int
	CPUUsage    float64
	MemoryUsage float64
	MemoryTotal uint64
	MemoryFree  uint64
	MemoryUsed  uint64
	DiskUsage   float64
	DiskTotal   uint64
	DiskFree    uint64
	DiskUsed    uint64
}

func getSystemMetrics() (*SystemMetrics, error) {
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err != nil {
		return nil, err
	}

	cpuCount, err := cpu.Counts(true)
	if err != nil {
		return nil, err
	}

	memoryUsage, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	diskUsage, err := disk.Usage("/")
	if err != nil {
		return nil, err
	}

	return &SystemMetrics{
		Timestamp:   time.Now(),
		CPUCount:    cpuCount,
		CPUUsage:    cpuPercent[0],
		MemoryUsage: memoryUsage.UsedPercent,
		MemoryTotal: memoryUsage.Total,
		MemoryFree:  memoryUsage.Free,
		MemoryUsed:  memoryUsage.Used,
		DiskUsage:   diskUsage.UsedPercent,
		DiskTotal:   diskUsage.Total,
		DiskFree:    diskUsage.Free,
		DiskUsed:    diskUsage.Used,
	}, nil
}

func saveSystemMetrics(metrics *SystemMetrics, fileName string) {
	if err := os.MkdirAll("./reports", 0755); err != nil {
		log.Printf("Error creating reports directory: %v\n", err)
		return
	}

	file, err := os.Create(fileName)
	if err != nil {
		log.Printf("Error creating metrics file: %v\n", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Write([]string{"Timestamp", "CPUCount", "CPUUsage", "MemoryUsage", "MemoryTotal", "MemoryFree", "MemoryUsed", "DiskUsage", "DiskTotal", "DiskFree", "DiskUsed"})
	err = writer.Write([]string{
		metrics.Timestamp.Format(time.RFC3339),
		fmt.Sprintf("%d", metrics.CPUCount),
		fmt.Sprintf("%f", metrics.CPUUsage),
		fmt.Sprintf("%f", metrics.MemoryUsage),
		fmt.Sprintf("%d", metrics.MemoryTotal),
		fmt.Sprintf("%d", metrics.MemoryFree),
		fmt.Sprintf("%d", metrics.MemoryUsed),
		fmt.Sprintf("%f", metrics.DiskUsage),
		fmt.Sprintf("%d", metrics.DiskTotal),
		fmt.Sprintf("%d", metrics.DiskFree),
		fmt.Sprintf("%d", metrics.DiskUsed),
	})
	if err != nil {
		log.Printf("Error writing to metrics file: %v\n", err)
		return
	}
	writer.Flush()
}

func sendMetricsToEmail(fileName string) {
	if err := godotenv.Load(); err != nil {
		log.Printf("Error loading .env file: %v\n", err)
		return
	}

	message := gomail.NewMessage()
	message.SetHeader("From", os.Getenv("SMTP_FROM"))
	message.SetHeader("To", os.Getenv("SMTP_TO"))
	message.SetHeader("Subject", "System Metrics")

	message.Attach(fileName)

	smtpPort, err := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if err != nil {
		log.Fatalf("Invalid SMTP port: %v", err)
	}

	dialer := gomail.NewDialer(
		os.Getenv("SMTP_HOST"),
		smtpPort,
		os.Getenv("SMTP_USER"),
		os.Getenv("SMTP_PASSWORD"),
	)

	maxRetries := 3
	retryDelay := 5 * time.Second

	for i := 0; i < maxRetries; i++ {
		if err := dialer.DialAndSend(message); err != nil {
			log.Printf("Attempt %d: Error sending email: %v\n", i+1, err)
			if i < maxRetries-1 {
				log.Printf("Retrying in %v...\n", retryDelay)
				time.Sleep(retryDelay)
			}
		} else {
			log.Println("Email sent successfully")
			return
		}
	}
}

func main() {
	currentTime := time.Now().Format("2006-01-02_150405")
	fileName := fmt.Sprintf("%smetrics_%s.csv", os.Getenv("DIRECTORY_PATH"), currentTime)

	metrics, err := getSystemMetrics()
	if err != nil {
		log.Printf("Error getting system metrics: %v\n", err)
		return
	}
	log.Printf("Metrics: %+v\n", metrics)
	saveSystemMetrics(metrics, fileName)

	log.Println("Sending metrics to email")
	sendMetricsToEmail(fileName)
}
