package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	mp "github.com/mackerelio/go-mackerel-plugin-helper"
)

var sock string

func parseAddress(uri string) (scheme, path string, err error) {
	u, err := url.Parse(uri)
	if err != nil {
		return scheme, path, err
	}

	scheme = u.Scheme
	path = u.Path
	if path == "" {
		path = u.Host
	}

	return scheme, path, err
}

func parseBody(r io.Reader) (stats map[string]interface{}, err error) {
	scanner := bufio.NewScanner(r)
	stats = make(map[string]interface{})
	for scanner.Scan() {
		p := strings.Split(scanner.Text(), " ")
		if len(p) == 2 {
			stats[strings.Trim(p[0], ":")], err = strconv.ParseFloat(p[1], 64)
		} else {
			stats[strings.Trim(p[len(p)-2], ":")], err = strconv.ParseFloat(p[len(p)-1], 64)
		}
	}

	return stats, err
}

func parseBodyHttp(uri string) (stats map[string]interface{}, err error) {
	var req *http.Request
	req, err = http.NewRequest("GET", uri, nil)
	if err != nil {
		return stats, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return stats, err
	}
	defer resp.Body.Close()

	stats, err = parseBody(resp.Body)

	return stats, err
}

func fakeDial(proto, addr string) (conn net.Conn, err error) {
	return net.Dial("unix", sock)
}

func parseBodyUnix(path string) (stats map[string]interface{}, err error) {
	tr := &http.Transport{
		Dial: fakeDial,
	}

	client := &http.Client{Transport: tr}
	resp, err := client.Get(fmt.Sprintf("http://dummy/%s", strings.TrimLeft(path, "/")))

	if err != nil {
		return stats, err
	}
	defer resp.Body.Close()

	stats, err = parseBody(resp.Body)

	return stats, err
}

type UnicornStatsPlugin struct {
	Address   string
	Path      string
	MetricKey string
}

// FetchMetrics interface for mackerelplugin
func (u UnicornStatsPlugin) FetchMetrics() (stats map[string]interface{}, err error) {
	stats, err = u.parseStats()
	return stats, err
}

func (u UnicornStatsPlugin) parseStats() (stats map[string]interface{}, err error) {
	scheme, path, err := parseAddress(u.Address)

	switch scheme {
	case "http":
		stats, err = parseBodyHttp(fmt.Sprintf("%s/%s", u.Address, strings.TrimLeft(u.Path, "/")))
	case "unix":
		sock = path
		stats, err = parseBodyUnix(u.Path)
	}

	return stats, err
}

// GraphDefinition interface for mackerelplugin
func (u UnicornStatsPlugin) GraphDefinition() map[string](mp.Graphs) {
	scheme, path, err := parseAddress(u.Address)
	if err != nil {
		log.Fatal(err)
	}

	var label string
	if u.MetricKey == "" {
		switch scheme {
		case "http":
			_, port, _ := net.SplitHostPort(path)
			u.MetricKey = port
			label = fmt.Sprintf("Unicorn Port %s Stats", port)
		case "unix":
			u.MetricKey = strings.Replace(strings.Replace(path, "/", "_", -1), ".", "_", -1)
			label = fmt.Sprintf("Unicorn %s Stats", path)
		}
	} else {
		label = fmt.Sprintf("Unicorn %s Stats", u.MetricKey)
	}

	return map[string](mp.Graphs){
		fmt.Sprintf("%s.unicorn.stats", u.MetricKey): mp.Graphs{
			Label: label,
			Unit:  "integer",
			Metrics: [](mp.Metrics){
				mp.Metrics{Name: "queued", Label: "Queued", Diff: false},
				mp.Metrics{Name: "active", Label: "Active", Diff: false},
				mp.Metrics{Name: "writing", Label: "Writing", Diff: false},
				mp.Metrics{Name: "calling", Label: "Calling", Diff: false},
			},
		},
	}
}

func main() {
	optAddress := flag.String("address", "http://localhost:8080", "URL or Unix Domain Socket")
	optPath := flag.String("path", "/_raindrops", "Path")
	optMetricKey := flag.String("metric-key", "", "Metric Key")
	optVersion := flag.Bool("version", false, "Version")
	optTempfile := flag.String("tempfile", "", "Temp file name")
	flag.Parse()

	if *optVersion {
		fmt.Println("0.2")
		os.Exit(0)
	}

	var unicorn UnicornStatsPlugin
	unicorn.Address = *optAddress
	unicorn.Path = *optPath
	unicorn.MetricKey = *optMetricKey

	helper := mp.NewMackerelPlugin(unicorn)
	if *optTempfile != "" {
		helper.Tempfile = *optTempfile
	} else {
		helper.Tempfile = fmt.Sprintf("/tmp/mackerel-plugin-unicorn-stats")
	}

	if os.Getenv("MACKEREL_AGENT_PLUGIN_META") != "" {
		helper.OutputDefinitions()
	} else {
		helper.OutputValues()
	}
}
