package network

import (
	"fmt"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/go-ping/ping"
)

// Stores the result of a ping request
type PingResult struct {
	// Ping Status
	Status bool
	// Ping Status Message
	Message string
	// Ping Address
	Addr string
	// Average time taken
	AvgTime time.Duration
}

// Stores results of PingAddrs
type PingResults struct {
	// Success Results
	Success []PingResult
	// Failed Results
	Failed []PingResult
}

// Pings an IP for "count" number of times
func PingIP(ip string, count int) (res PingResult, err error) {
	res.Addr = ip

	pinger, err := ping.NewPinger(ip)
	if err != nil {
		res.Message = err.Error()
		return
	}

	pinger.Count = count
	err = pinger.Run()
	if err != nil {
		res.Message = err.Error()
		return
	}
	res.Status = true
	res.AvgTime = pinger.Statistics().AvgRtt

	return
}

// Ping a list of addresses and sort the results in asc or dsc order
func PingAddrs(addresses []string, asc bool) (result PingResults) {
	wg := new(sync.WaitGroup)
	mutex := new(sync.Mutex)
	result = PingResults{
		Success: []PingResult{},
		Failed:  []PingResult{},
	}

	for _, addr := range addresses {
		ipAddr, err := net.ResolveIPAddr("ip", addr)
		if err != nil {
			res := PingResult{
				Status:  false,
				Addr:    addr,
				Message: fmt.Sprintf("Unable to resolve: %v", err.Error()),
			}
			result.Failed = append(result.Failed, res)
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := PingIP(ipAddr.IP.String(), 3)

			mutex.Lock()
			defer mutex.Unlock()
			if err != nil {
				result.Failed = append(result.Failed, res)
			} else {
				result.Success = append(result.Success, res)
			}
		}()
	}
	wg.Wait()

	sort.Slice(result.Success, func(i, j int) bool {
		diff := result.Success[i].AvgTime - result.Success[j].AvgTime
		if asc {
			return diff <= 0
		}
		return diff > 0
	})

	return
}
