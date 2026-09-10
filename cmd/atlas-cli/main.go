package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var version = "0.1.0"

type flags struct {
	values map[string]string
}

// parseFlags parses "--key value" (or "--key=value") pairs.
func parseFlags(args []string) flags {
	f := flags{values: make(map[string]string)}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			continue
		}
		key := strings.TrimPrefix(arg, "--")
		value := "true"
		if eq := strings.Index(key, "="); eq >= 0 {
			value = key[eq+1:]
			key = key[:eq]
		} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			value = args[i+1]
			i++
		}
		f.values[key] = value
	}
	return f
}

func (f flags) get(key string) string {
	return f.values[key]
}

func (f flags) has(key string) bool {
	_, ok := f.values[key]
	return ok
}

func (f flags) int(key string) int {
	v := f.get(key)
	if v == "" {
		return 0
	}
	n, _ := strconv.Atoi(v)
	return n
}

func (f flags) duration(key string) time.Duration {
	v := f.get(key)
	if v == "" {
		return 0
	}
	d, _ := time.ParseDuration(v)
	return d
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	apiBase := os.Getenv("ATLAS_API_URL")
	if apiBase == "" {
		apiBase = "http://localhost:8080"
	}
	do := newClient(apiBase)

	var err error
	switch cmd {
	case "version":
		fmt.Printf("atlas %s\n", version)
	case "status":
		getJSON(do, "/api/v1/health", "Health")
		getJSON(do, "/api/v1/info", "Info")
	case "nodes":
		f := parseFlags(args)
		switch {
		case len(args) > 0 && args[0] == "add":
			err = requireFlags(f, "name")
			if err == nil {
				err = do.postJSON("/api/v1/nodes", map[string]interface{}{
					"id":      firstNonEmpty(f.get("id"), f.get("name")),
					"name":    f.get("name"),
					"type":    firstNonEmpty(f.get("type"), "server"),
					"state":   firstNonEmpty(f.get("state"), "healthy"),
					"health":  floatVal(f.get("health"), 1.0),
					"cpu":     floatVal(f.get("cpu"), 0),
					"memory":  floatVal(f.get("memory"), 0),
					"disk":    floatVal(f.get("disk"), 0),
					"network": floatVal(f.get("network"), 0),
					"load":    floatVal(f.get("load"), 0),
				})
			}
		case len(args) > 0 && args[0] == "rm":
			if len(args) < 2 {
				err = fmt.Errorf("usage: atlas nodes rm <id>")
			} else {
				err = do.delete("/api/v1/nodes/" + args[1])
			}
		default:
			getJSON(do, "/api/v1/nodes", "Nodes")
		}
	case "topology":
		getJSON(do, "/api/v1/topology", "Topology")
	case "services":
		getJSON(do, "/api/v1/services", "Services")
	case "jobs":
		f := parseFlags(args)
		if len(args) > 0 && args[0] == "create" {
			err = requireFlags(f, "type")
			if err == nil {
				job := map[string]interface{}{
					"type":     f.get("type"),
					"priority": f.int("priority"),
					"payload":  f.get("payload"),
				}
				if f.has("max_retries") {
					job["max_retries"] = f.int("max_retries")
				}
				err = do.postJSON("/api/v1/jobs", job)
			}
		} else {
			getJSON(do, "/api/v1/jobs", "Jobs")
		}
	case "cluster":
		getJSON(do, "/api/v1/cluster", "Cluster")
	case "raft":
		getJSON(do, "/api/v1/raft", "Raft")
	case "metrics":
		getJSON(do, "/api/v1/metrics", "Metrics")
	case "chaos":
		f := parseFlags(args)
		switch {
		case len(args) > 0 && args[0] == "run":
			err = requireFlags(f, "type", "target")
			if err == nil {
				d := f.duration("duration")
				exp := map[string]interface{}{
					"id":       firstNonEmpty(f.get("id"), "exp-"+strconv.FormatInt(time.Now().UnixNano(), 10)),
					"type":     f.get("type"),
					"target":   f.get("target"),
					"duration": d.Nanoseconds(),
					"config":   map[string]interface{}{},
				}
				err = do.postJSON("/api/v1/chaos", exp)
			}
		case len(args) > 0 && args[0] == "stop":
			if len(args) < 2 {
				err = fmt.Errorf("usage: atlas chaos stop <id>")
			} else {
				err = do.postJSON("/api/v1/chaos/"+args[1]+"/stop", map[string]interface{}{})
			}
		default:
			getJSON(do, "/api/v1/chaos", "Chaos experiments")
		}
	case "incidents":
		switch {
		case len(args) > 0 && args[0] == "analyze":
			if len(args) < 2 {
				err = fmt.Errorf("usage: atlas incidents analyze <id>")
			} else {
				err = do.post("/api/v1/incidents/"+args[1]+"/analyze", "Analysis")
			}
		case len(args) > 0 && args[0] == "resolve":
			if len(args) < 2 {
				err = fmt.Errorf("usage: atlas incidents resolve <id> --reason \"...\"")
			} else {
				f := parseFlags(args[2:])
				err = do.postJSON("/api/v1/incidents/"+args[1]+"/resolve",
					map[string]interface{}{"resolution": f.get("reason")})
			}
		case len(args) > 0 && args[0] == "investigate":
			if len(args) < 2 {
				err = fmt.Errorf("usage: atlas incidents investigate <id>")
			} else {
				err = do.postJSON("/api/v1/incidents/"+args[1]+"/investigate", map[string]interface{}{})
			}
		default:
			if len(args) > 0 && args[0] == "--open" {
				getJSON(do, "/api/v1/incidents?open=true", "Open incidents")
			} else {
				getJSON(do, "/api/v1/incidents", "Incidents")
			}
		}
	case "anomalies":
		f := parseFlags(args)
		switch {
		case len(args) > 0 && args[0] == "ingest":
			err = requireFlags(f, "node", "metric", "value")
			if err == nil {
				err = do.postJSON("/api/v1/anomalies/ingest", map[string]interface{}{
					"node_id": f.get("node"),
					"metric":  f.get("metric"),
					"value":   floatVal(f.get("value"), 0),
				})
			}
		case len(args) > 0 && args[0] == "rule":
			err = requireFlags(f, "metric")
			if err == nil {
				rule := map[string]interface{}{"metric": f.get("metric")}
				if f.has("high") {
					rule["critical_high"] = floatVal(f.get("high"), 0)
				}
				if f.has("low") {
					rule["critical_low"] = floatVal(f.get("low"), 0)
				}
				err = do.postJSON("/api/v1/anomalies/rules", rule)
			}
		default:
			getJSON(do, "/api/v1/anomalies", "Anomalies")
		}
	case "benchmarks":
		switch {
		case len(args) > 0 && args[0] == "names":
			getJSON(do, "/api/v1/benchmarks/name", "Available benchmarks")
		case len(args) > 0 && args[0] == "run":
			if len(args) < 2 {
				err = fmt.Errorf("usage: atlas benchmarks run <name|all>")
			} else if args[1] == "all" {
				err = do.post("/api/v1/benchmarks/all", "Running all benchmarks")
			} else {
				err = do.post("/api/v1/benchmarks/"+args[1], "Benchmark "+args[1])
			}
		default:
			getJSON(do, "/api/v1/benchmarks", "Benchmarks")
		}
	case "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Atlas CLI — Distributed Infrastructure Intelligence Engine

Usage:
  atlas <command> [commands] [--options]

Environment:
  ATLAS_API_URL     API base URL (default http://localhost:8080)

Commands:
  status                      Show system health and info
  version                     Show version
  nodes                       List infrastructure nodes
  nodes add --name <n> [--type server] [--health 1.0]
  nodes rm <id>               Delete a node
  topology                    Show infrastructure topology graph
  services                    List services
  jobs                        List jobs
  jobs create --type <T> [--priority <n>] [--payload '{"k":"v"}']
  cluster                     Show cluster status
  raft                        Show Raft consensus status
  metrics                     Show metrics registry
  anomalies                   List anomalies
  anomalies ingest --node <id> --metric <m> --value <v>
  anomalies rule --metric <m> [--high 90] [--low 5]
  chaos                       List chaos experiments
  chaos run --type kill_node|... --target <node> [--duration 30s]
  chaos stop <id>             Stop and revert a running experiment
  incidents [--open]          List incidents (or only open ones)
  incidents investigate <id>
  incidents resolve <id> --reason "..."
  incidents analyze <id>      Run advisory AI analysis
  benchmarks                  List benchmark results
  benchmarks run <name|all>   Run one benchmark or the full set
  benchmarks names            List available benchmark names
  help                        Show this help`)
}

func requireFlags(f flags, keys ...string) error {
	for _, k := range keys {
		if !f.has(k) {
			return fmt.Errorf("missing required option --%s", k)
		}
	}
	return nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func floatVal(s string, def float64) float64 {
	if s == "" {
		return def
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

type client struct {
	base   string
	http   *http.Client
}

func newClient(base string) *client {
	return &client{base: base, http: &http.Client{Timeout: 5 * time.Second}}
}

func (c *client) do(method, path string, body interface{}, label string) error {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode body: %w", err)
		}
		reader = bytes.NewReader(buf)
	}
	req, err := http.NewRequest(method, c.base+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var data interface{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &data); err != nil {
			data = string(raw)
		}
	}
	if label != "" {
		fmt.Printf("--- %s ---\n", label)
	}
	tryPrettyPrint(data)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("API %s failed: %s", resp.Status, statusMessage(data))
	}
	return nil
}

func (c *client) get(path, label string) error {
	return c.do(http.MethodGet, path, nil, label)
}

func (c *client) post(path, label string) error {
	return c.do(http.MethodPost, path, nil, label)
}

func (c *client) postJSON(path string, body interface{}) error {
	return c.do(http.MethodPost, path, body, "")
}

func (c *client) delete(path string) error {
	return c.do(http.MethodDelete, path, nil, "")
}

func tryPrettyPrint(data interface{}) {
	if data == nil {
		return
	}
	if s, ok := data.(string); ok {
		fmt.Println(s)
		return
	}
	pretty, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Println(data)
		return
	}
	fmt.Println(string(pretty))
}

func statusMessage(data interface{}) string {
	m, ok := data.(map[string]interface{})
	if !ok {
		return ""
	}
	if e, ok := m["error"].(string); ok {
		return e
	}
	if e, ok := m["message"].(string); ok {
		return e
	}
	return ""
}

func getJSON(c *client, path, label string) {
	if err := c.get(path, label); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}