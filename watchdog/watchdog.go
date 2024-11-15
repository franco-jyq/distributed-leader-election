package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	isLeader          bool
	nodeID            int
	leaderID          int
	nodes             = []int{1, 2, 3} // List of node IDs
	isLeaderMutex     sync.Mutex
	mutex             sync.Mutex
	heartbeatInterval = 5 * time.Second
	heartbeatTimeout  = 2 * time.Second
)

func main() {
	// Get the node ID from the environment variable
	id, err := strconv.Atoi(os.Getenv("WATCHDOG_HOST"))
	if err != nil {
		fmt.Println("Invalid node ID")
		return
	}
	nodeID = id

	// Start the server in a goroutine
	go startServer()

	// Wait a moment to ensure the server is running
	time.Sleep(1 * time.Second)

	// Start the leader election process
	startElection()

	// Iniciar el proceso de heartbeat en una goroutine
	go startHeartbeat()

	// Start the leader task in a goroutine
	go performLeaderTask()

	// Keep the program running
	select {}
}

func startServer() {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", 8000+nodeID))
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer ln.Close()
	fmt.Printf("Node %d listening on port %d\n", nodeID, 8000+nodeID)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading data:", err)
		return
	}
	fields := strings.Split(string(buf[:n]), " ")
	message := fields[0]
	fmt.Printf("Node %d received message: %s\n", nodeID, message)

	if message == "ELECTION" {
		// Respond to the election message
		conn.Write([]byte("OK"))
		startElection()
	} else if message == "COORDINATOR" {
		// Update the leader ID
		leaderID, err = strconv.Atoi(fields[1])
		if err != nil {
			fmt.Println("Error parsing leader ID:", err)
			return
		}
		isLeaderMutex.Lock()
		isLeader = (leaderID == nodeID)
		isLeaderMutex.Unlock()
		fmt.Printf("Node %d recognizes node %d as leader\n", nodeID, leaderID)
	} else if message == "HEARTBEAT" {
		// Respond to the heartbeat message
		conn.Write([]byte("ALIVE"))
	}
}

func startElection() {
	mutex.Lock()
	defer mutex.Unlock()

	fmt.Printf("Node %d starting election\n", nodeID)
	isLeaderMutex.Lock()
	isLeader = true
	isLeaderMutex.Unlock()

	for _, id := range nodes {
		if id > nodeID {
			isLeaderMutex.Lock()
			isLeader = false
			isLeaderMutex.Unlock()
			fmt.Printf("Dialing to port 800%d\n", id)
			conn, err := net.Dial("tcp", fmt.Sprintf("watchdog_%d:800%d", id, id))
			if err != nil {
				fmt.Printf("Node %d could not connect to node %d which is in port 800%d\n", nodeID, id, id)
				isLeaderMutex.Lock()
				isLeader = true
				isLeaderMutex.Unlock()
				continue
			}
			defer conn.Close()
			conn.Write([]byte("ELECTION"))
			buf := make([]byte, 1024)
			conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			n, err := conn.Read(buf)
			if err == nil && string(buf[:n]) == "OK" {
				fmt.Printf("Node %d received OK from node %d\n", nodeID, id)
				return
			}
		}
	}

	isLeaderMutex.Lock()
	if isLeader {
		leaderID = nodeID
		fmt.Printf("Node %d is the new leader\n", nodeID)
		for _, id := range nodes {
			if id != nodeID {
				conn, err := net.Dial("tcp", fmt.Sprintf("watchdog_%d:800%d", id, id))
				if err != nil {
					fmt.Printf("Node %d could not connect to node %d\n", nodeID, id)
					continue
				}
				defer conn.Close()
				conn.Write([]byte(fmt.Sprintf("COORDINATOR %d", nodeID)))
			}
		}
	}
	isLeaderMutex.Unlock()
}

func startHeartbeat() {
	for {
		time.Sleep(heartbeatInterval)
		isLeaderMutex.Lock()
		if isLeader {
			isLeaderMutex.Unlock()
			continue
		}
		isLeaderMutex.Unlock()
		conn, err := net.Dial("tcp", fmt.Sprintf("watchdog_%d:800%d", leaderID, leaderID))
		if err != nil {
			fmt.Printf("Node %d could not connect to leader %d\n", nodeID, leaderID)
			startElection()
			continue
		}
		defer conn.Close()
		conn.SetWriteDeadline(time.Now().Add(heartbeatTimeout))
		_, err = conn.Write([]byte("HEARTBEAT"))
		if err != nil {
			fmt.Printf("Node %d did not receive heartbeat response from leader %d\n", nodeID, leaderID)
			startElection()
		}
	}
}

func performLeaderTask() {
	for {
		time.Sleep(10 * time.Second) // Adjust the interval as needed
		isLeaderMutex.Lock()
		if isLeader {
			fmt.Printf("Node %d is performing the leader task\n", nodeID)
			if !pingHost("dummy:80") {
				fmt.Println("dummy-program is not responding, restarting service...")
				restartService()
			}
			fmt.Println("Ping successful")
		}
		isLeaderMutex.Unlock()
	}
}

func pingHost(host string) bool {
	conn, err := net.DialTimeout("tcp", host, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func restartService() {
	cmd := exec.Command("docker", "start", "dummy")
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error restarting service: %v\n", err)
	} else {
		fmt.Println("Service restarted successfully")
	}
}
