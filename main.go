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
	"sync"
	"sync/atomic"
	"time"
)

type Result struct {
	ProxyAddress     string
	DestAddress      string
	Timeout          int
	MaxAsyncRequests int
	mu               *sync.Mutex
	Error            []*Response
	Success          []*Response
}

type Response struct {
	Port      string
	IsSuccess bool
	Error     error
	Response  string
}

type ResponseCheckIP struct {
	Status bool
	Reason string
	Data   ResponseCheckIPData
}

type ResponseCheckIPData struct {
	Carrier     string
	City        string
	CountryCode string
	CountryName string
	Ip          string
	Isp         string
	Region      string
}

func main() {
	// Register flags.
	proxyAddressExternal := flag.String("proxy-host", "", "socks5://host | http://host | https://host")
	portFrom := flag.Int("proxy-port-from", 0, "Ex.: 17000")
	portTo := flag.Int("proxy-port-to", 0, "Ex.: 17999")
	destAddressExternal := flag.String("dest", "https://checker.soax.com/api/ipinfo", "count of iterations")
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
	responseChan := make(chan *Response, *asyncChannels)

	result := &Result{
		ProxyAddress:     *proxyAddressExternal,
		DestAddress:      *destAddressExternal,
		Timeout:          *timeoutSeconds,
		MaxAsyncRequests: *asyncChannels,
		mu:               &sync.Mutex{},
		Error:            make([]*Response, 0),
		Success:          make([]*Response, 0),
	}

	fmt.Println("Starting...", "proxy:", *proxyAddressExternal, "from:", *portFrom, "to:", *portTo)

	for port := *portFrom; port <= *portTo; port++ {
		go sendRequest(proxyAddressExternal, timeoutSeconds, port, destUrl, responseChan)
	}

	for {
		select {
		case <-done:
			ipList := getIPCounter(result.Success)
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

func sendRequest(proxyAddressExternal *string, timeoutSeconds *int, port int, destUrl *url.URL, responseChan chan *Response) {
	proxyUrl, err := url.Parse(*proxyAddressExternal + ":" + strconv.Itoa(port))
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
		defer resp.Body.Close()

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

func createResponseToChan(port int, isSuccess bool, err error, response string) *Response {
	return &Response{
		Port:      strconv.Itoa(port),
		IsSuccess: isSuccess,
		Error:     err,
		Response:  response,
	}
}

func getIPCounter(responses []*Response) []*Response {
	dublicate := make(map[string]int, 0)

	response := &ResponseCheckIP{}

	// Find
	for _, r := range responses {
		err := json.Unmarshal([]byte(r.Response), response)
		if err == nil {
			if count, ok := dublicate[response.Data.Ip]; !ok {
				dublicate[response.Data.Ip] = 1
			} else {
				dublicate[response.Data.Ip] = count + 1
			}
		} else {
			fmt.Println("error on Unmarshal response: ", err)
		}
	}

	result := make([]*Response, 0)

	// for
	for ip, count := range dublicate {
		result = append(result, &Response{
			Port:     fmt.Sprintf(ip + "            ")[0:15],
			Response: strconv.Itoa(count),
		})
	}

	return result
}

func (r *Result) createReport(responses []*Response, cellOneName string, fileNameLogs string) {
	f, err := os.OpenFile(fileNameLogs, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("error opening file:", err.Error())
	} else {
		defer f.Close()
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
