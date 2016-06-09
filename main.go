package main

import (
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
	// "strings"
)

const (
	EXCLUDE_ENDPOINT   float32 = float32(0)
	AUTO_FREQ_ENDPOINT float32 = float32(-1)
)

var (
	numberOfRequests   int64
	concurrencyLevel   int64
	configFile         string
	endpoint           string
	httpTransport      *http.Transport
	client             *http.Client
	random             *rand.Rand
	endpointLookup     map[string]segment
	resultDistribution map[string]endpointStats
)

type (
	Endpoint struct {
		Path string
		// NOTE(john): -1 = auto, 0 = exclude, > 0 = percentage
		Freq float32
	}
	Config struct {
		Root      string
		Endpoints []Endpoint
	}
	segment struct {
		start float32
		stop  float32
	}
	request struct {
		url string
		id  int64
	}
	response struct {
		url      string
		id       int64
		duration time.Duration
		pass     bool
	}
	endpointStats struct {
		url      string
		max      int64
		min      int64
		avg      float64
		count    int64
		failures int64
		freq     float32
	}
)

func init() {
	flag.Int64Var(&numberOfRequests, "n", 0, "The total number of requests to make.")
	flag.Int64Var(&concurrencyLevel, "c", 1, "The number of concurrent requests to execute.")
	flag.StringVar(&configFile, "config", "./config.yml", "File path to a configuration file (yml)")
	flag.Parse()
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func isValidUrl(raw string) bool {
	_, err := url.ParseRequestURI(raw)
	return err == nil
}

func readConfig(path string) Config {
	data, readFileErr := ioutil.ReadFile(path)
	if readFileErr != nil {
		panic(readFileErr)
	}

	config := Config{}
	yamlUnmarshalErr := yaml.Unmarshal(data, &config)
	if yamlUnmarshalErr != nil {
		panic(yamlUnmarshalErr)
	}
	return config
}

func makeRequest(done chan response, req request) {
	requestStartTime := time.Now()
	resp, err := client.Get(req.url)
	if err != nil {
		fmt.Println(err)
		done <- response{id: req.id, url: req.url, duration: time.Since(requestStartTime), pass: false}
		return
	}
	defer resp.Body.Close()
	io.Copy(ioutil.Discard, resp.Body)
	done <- response{id: req.id, url: req.url, duration: time.Since(requestStartTime), pass: resp.StatusCode == 200 || resp.StatusCode == 304}
}

func buildUrlMap(config Config) map[string]segment {
	part := float32(1.0)
	leftovers := 0
	for _, endpoint := range config.Endpoints {
		if endpoint.Freq != EXCLUDE_ENDPOINT && endpoint.Freq != AUTO_FREQ_ENDPOINT {
			part -= endpoint.Freq * 0.01
		} else if endpoint.Freq == AUTO_FREQ_ENDPOINT {
			leftovers++
		}
	}
	distribution := part / float32(leftovers)
	if part < 0 {
		panic("Unbalanced frequencies!")
	}
	position := float32(0)
	lookup := make(map[string]segment)
	for _, endpoint := range config.Endpoints {
		if endpoint.Freq == EXCLUDE_ENDPOINT {
			continue
		}
		segment := segment{start: position}
		var freq float32
		if endpoint.Freq == AUTO_FREQ_ENDPOINT {
			position += distribution
			freq = distribution
		} else {
			position += endpoint.Freq * 0.01
			freq = endpoint.Freq * 0.01
		}
		segment.stop = position
		endpointUrl := config.Root + endpoint.Path
		lookup[endpointUrl] = segment
		resultDistribution[endpointUrl] = endpointStats{
			url:  endpointUrl,
			freq: freq,
			min:  int64(math.MaxInt64),
			max:  int64(0),
		}
	}
	return lookup
}

func randomUrl() string {
	selection := random.Float32()
	for url, segment := range endpointLookup {
		if segment.start < selection && segment.stop > selection {
			return url
		}
	}
	return ""
}

// TODO(john): if n = 0, just start making requests and output count every 30 seconds?
func main() {
	httpTransport = &http.Transport{}
	client = &http.Client{Transport: httpTransport}
	startTime := time.Now()
	fmt.Println("Starting thrashing:", startTime)
	lastArg := os.Args[len(os.Args)-1]
	smoothing := float64(0.7)

	queue := make(chan request, concurrencyLevel*2)
	completions := make(chan response, concurrencyLevel)
	resultDistribution = make(map[string]endpointStats)

	for page := int64(0); page < concurrencyLevel; page++ {
		go func() {
			for req := range queue {
				makeRequest(completions, req)
			}
		}()
	}

	if isValidUrl(lastArg) {
		resultDistribution[lastArg] = endpointStats{
			url:  lastArg,
			freq: 1.0,
			min:  int64(math.MaxInt64),
			max:  int64(0),
		}
		go func() {
			for i := int64(0); i < numberOfRequests; i++ {
				queue <- request{url: lastArg, id: i}
			}
		}()
	} else {
		config := readConfig(configFile)
		endpointLookup = buildUrlMap(config)
		go func() {
			for i := int64(0); i < numberOfRequests; i++ {
				url := randomUrl()
				queue <- request{url: url, id: i}
			}
		}()
	}

	count := int64(1)
	interval := numberOfRequests / 10
	if interval < 1 {
		interval = 1
	}
	checkin := time.Now()
	for current := range completions {
		stats := resultDistribution[current.url]
		if int64(current.duration) < stats.min {
			stats.min = int64(current.duration)
		}
		if int64(current.duration) > stats.max {
			stats.max = int64(current.duration)
		}
		stats.avg = (stats.avg * smoothing) + (current.duration.Seconds() * (1.0 - smoothing))
		stats.count++
		if !current.pass {
			stats.failures++
		}
		resultDistribution[current.url] = stats
		if count%interval == 0 || count == numberOfRequests {
			fmt.Print("Completed requests: ", count)
			fmt.Print("\t", time.Since(startTime), "\t")
			fmt.Print("Req/Sec: ")
			fmt.Printf("%v", strconv.FormatFloat(float64(interval)/time.Since(checkin).Seconds(), 'f', -1, 32))
			fmt.Print("\n")
			checkin = time.Now()
		}
		count++
		if count > numberOfRequests {
			break
		}
	}

	duration := time.Since(startTime)
	totalFailures := int64(0)
	fmt.Println("")
	fmt.Println("Request Summaries")
	fmt.Println("=================================")
	for _, stats := range resultDistribution {
		fmt.Println("")
		totalFailures += stats.failures
		fmt.Println(stats.url)
		fmt.Println("  Count:  ", stats.count)
		fmt.Println("  Success:", (1.0-(float64(stats.failures)/float64(stats.count)))*100, "%")
		fmt.Println("  Min:    ", time.Duration(stats.min).Seconds()*1000, "ms")
		fmt.Println("  Max:    ", time.Duration(stats.max).Seconds()*1000, "ms")
		fmt.Println("  Avg:    ", time.Duration(stats.avg), "ms")
		if stats.freq < 1.0 {
			fmt.Println("  Freq:   ", stats.freq*100, "%")
		}
	}
	fmt.Println("")
	fmt.Println("Result Summary")
	fmt.Println("=================================")
	fmt.Println("  Req Count:  ", numberOfRequests)
	fmt.Println("  Concurrency:", concurrencyLevel)
	fmt.Println("  Duration:   ", duration)
	fmt.Println("  Success:    ", (1.0-(float64(totalFailures)/float64(numberOfRequests)))*100, "%")
	fmt.Println("  Req/Sec:    ", float64(numberOfRequests)/duration.Seconds())
}
