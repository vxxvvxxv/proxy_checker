package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type resultProxy struct {
	ProxyAddress     string
	DestAddress      string
	Timeout          int
	MaxAsyncRequests int
	mu               *sync.Mutex
	Error            []*resultResponse
	Success          []*resultResponse
}

type resultResponse struct {
	Port      string
	IsSuccess bool
	Error     error
	Response  string
}

type resultDuplicatesIpPort map[string]int

const labelPort = "%PORT%"

func main() {
	// Register flags.
	proxyAddressExternal := flag.String("proxy-host", "", "socks5://host | http://host | https://host")
	portFrom := flag.Int("proxy-port-from", 0, "Ex.: 17000")
	portTo := flag.Int("proxy-port-to", 0, "Ex.: 17999")
	destAddressExternal := flag.String("dest", "https://ip.nf/me.json", "count of iterations")
	asyncChannels := flag.Int("async", 100, "count of async requests")
	timeoutSeconds := flag.Int("timeout", 60, "seconds of timeout request")
	reports := flag.Bool("reports", false, "save results in reports")
	// Parse the flags.
	flag.Parse()

	destUrl, err := url.Parse(*destAddressExternal)
	if err != nil {
		panic(err)
	}

	var countRequest int64 = 0
	done := make(chan struct{}, 1)
	responseChan := make(chan *resultResponse, *asyncChannels)

	result := &resultProxy{
		ProxyAddress:     *proxyAddressExternal,
		DestAddress:      *destAddressExternal,
		Timeout:          *timeoutSeconds,
		MaxAsyncRequests: *asyncChannels,
		mu:               &sync.Mutex{},
		Error:            make([]*resultResponse, 0),
		Success:          make([]*resultResponse, 0),
	}

	fmt.Println("Starting...", "proxy:", *proxyAddressExternal, "from:", *portFrom, "to:", *portTo)

	for port := *portFrom; port <= *portTo; port++ {
		go sendRequest(proxyAddressExternal, timeoutSeconds, port, destUrl, responseChan)
	}

	for {
		select {
		case <-done:
			ipList := getIPCounter(destUrl.String(), result.Success)
			fmt.Printf("Success: %d | Error: %d | Count uniq IP: %d\n", len(result.Success), len(result.Error), len(ipList))

			if *reports {
				year, month, day := time.Now().Date()

				fmt.Println("Preparing reports...")

				nameFile := fmt.Sprintf("log-%d-%d-%d_%d_%d", year, month, day, *portFrom, *portTo)
				nameSuccess := fmt.Sprintf("%s_success.log", nameFile)
				nameError := fmt.Sprintf("%s_error.log", nameFile)
				nameIP := fmt.Sprintf("%s_ip.log", nameFile)

				result.createReport(result.Success, "PORT", nameSuccess)
				result.createReport(result.Error, "PORT", nameError)
				result.createReport(ipList, "IP                 ", nameIP)

				fmt.Printf("Success requests saved in: %s\n", nameSuccess)
				fmt.Printf("Error requests saved in: %s\n", nameError)
				fmt.Printf("Uniq IP and count dublicates saved in: %s\n", nameIP)
			}

			fmt.Println("Done!")
			return
		case response := <-responseChan:

			result.mu.Lock()
			if response.IsSuccess {
				result.Success = append(result.Success, response)
			} else {
				result.Error = append(result.Error, response)
			}
			result.mu.Unlock()

			atomic.AddInt64(&countRequest, 1)
			count := atomic.LoadInt64(&countRequest)

			if count == int64((*portTo-*portFrom)+1) {
				done <- struct{}{}
			}
		}
	}
}

func sendRequest(proxyAddressExternal *string, timeoutSeconds *int, port int, destUrl *url.URL, responseChan chan *resultResponse) {
	prepareURLString := strings.ReplaceAll(*proxyAddressExternal, labelPort, strconv.Itoa(port))

	proxyUrl, err := url.Parse(prepareURLString)
	if err != nil {
		body := fmt.Sprintf("Error on create address proxy: %s", err.Error())
		responseChan <- createResponseToChan(port, false, err, body)
		return
	}

	clientProxy := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)},
		Timeout:   time.Duration(*timeoutSeconds) * time.Second,
	}

	resp, err := clientProxy.Get(destUrl.String())
	if err != nil {
		body := fmt.Sprintf("Error on get request: %s", err.Error())
		responseChan <- createResponseToChan(port, false, err, body)
		return
	} else {
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			body := fmt.Sprintf("Error on check StatusCode: %d", resp.StatusCode)
			responseChan <- createResponseToChan(port, false, errors.New(body), body)
			return
		}

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			body := fmt.Sprintf("Error on get body: %s", err.Error())
			responseChan <- createResponseToChan(port, false, err, body)
			return
		}

		bodyString := string(bodyBytes)
		responseChan <- createResponseToChan(port, true, nil, bodyString)
		return
	}
}

func createResponseToChan(port int, isSuccess bool, err error, response string) *resultResponse {
	return &resultResponse{
		Port:      strconv.Itoa(port),
		IsSuccess: isSuccess,
		Error:     err,
		Response:  response,
	}
}

func getIPCounter(destAddr string, responses []*resultResponse) []*resultResponse {
	duplicatesList := getCounterIpByChecker(destAddr, responses)
	result := make([]*resultResponse, 0)

	// for
	for ip, count := range duplicatesList {
		result = append(result, &resultResponse{
			Port:     fmt.Sprintf(ip + "            ")[0:15],
			Response: strconv.Itoa(count),
		})
	}

	return result
}

func (r *resultProxy) createReport(responses []*resultResponse, cellOneName string, fileNameLogs string) {
	f, err := os.OpenFile(fileNameLogs, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("error opening file:", err.Error())
	} else {
		defer func() {
			_ = f.Close()
		}()
		log.SetOutput(f)
	}

	log.Println("Report for proxy")
	log.Println(getLine())
	log.Println("Proxy IP: ", r.ProxyAddress)
	log.Println("Dest address: ", r.DestAddress)
	log.Println("Timeout requests: ", r.Timeout)
	log.Println("Max async requests: ", r.MaxAsyncRequests)
	log.Println(getLine())
	log.Println("Count responses: ", len(responses))
	log.Println(getLine())
	log.Printf("%s    | RESULT\n", cellOneName)
	log.Println(getLine())

	for _, response := range responses {
		log.Printf("%s   | %s\n", response.Port, response.Response)
	}

	log.Println(getLine())
}

func getLine() string {
	return "--------------------------------------------------------------------"
}

// ---------------------------------------------------------------------------------------------------------------------

func getCounterIpByChecker(destAddr string, responses []*resultResponse) resultDuplicatesIpPort {
	switch destAddr {
	case "https://checker.soax.com/api/ipinfo":
		return getDuplicatesCheckerSoaxCom(responses)
	case "https://ip.nf/me.json":
		return getDuplicatesCheckerIpNf(responses)
	default:
		return make(resultDuplicatesIpPort, 0)
	}
}

// ---------------------------------------------------------------------------------------------------------------------

// ResponseCheckerSoaxCom https://checker.soax.com/api/ipinfo
type ResponseCheckerSoaxCom struct {
	Status bool
	Reason string
	Data   ResponseCheckerSoaxComData
}
type ResponseCheckerSoaxComData struct {
	Carrier     string
	City        string
	CountryCode string
	CountryName string
	Ip          string
	Isp         string
	Region      string
}

func getDuplicatesCheckerSoaxCom(responses []*resultResponse) resultDuplicatesIpPort {
	duplicatesList := make(resultDuplicatesIpPort, 0)

	response := &ResponseCheckerSoaxCom{}

	// Find
	for _, r := range responses {
		err := json.Unmarshal([]byte(r.Response), response)
		if err == nil {
			if count, ok := duplicatesList[response.Data.Ip]; !ok {
				duplicatesList[response.Data.Ip] = 1
			} else {
				duplicatesList[response.Data.Ip] = count + 1
			}
		} else {
			fmt.Println("error on Unmarshal response: ", err)
		}
	}

	return duplicatesList
}

// ---------------------------------------------------------------------------------------------------------------------

// ResponseIpNf https://ip.nf/me.json
type ResponseIpNf struct {
	Ip ResponseIpNfIp `json:"ip"`
}

type ResponseIpNfIp struct {
	Ip          string  `json:"ip"`
	Asn         string  `json:"asn"`
	Netmask     int     `json:"netmask"`
	Hostname    string  `json:"hostname"`
	City        string  `json:"city"`
	PostCode    string  `json:"post_code"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

func getDuplicatesCheckerIpNf(responses []*resultResponse) resultDuplicatesIpPort {
	duplicatesList := make(resultDuplicatesIpPort, 0)

	response := &ResponseIpNf{}

	// Find
	for _, r := range responses {
		err := json.Unmarshal([]byte(r.Response), response)
		if err == nil {
			if count, ok := duplicatesList[response.Ip.Ip]; !ok {
				duplicatesList[response.Ip.Ip] = 1
			} else {
				duplicatesList[response.Ip.Ip] = count + 1
			}
		} else {
			fmt.Println("error on Unmarshal response: ", err)
		}
	}

	return duplicatesList
}

// ---------------------------------------------------------------------------------------------------------------------
