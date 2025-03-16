package main

import (
	"embed"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"text/template"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	neti "github.com/shirou/gopsutil/v4/net"
)

const (
	responseEnvKey = "RESPONSE_TEXT"
	portEnvKey     = "PORT"
	exitEnvKey     = "DEBUG_EXIT"
	realIPHeader   = "X-Forwarded-For"
	defaultPort    = 8080
)

//go:embed index.html.tmpl
var content embed.FS

func main() {
	if val, ok := os.LookupEnv(exitEnvKey); ok {
		code, err := strconv.Atoi(val)
		if err != nil {
			log.Fatalf("error parsing exit code %s to int: %s", val, err)
		}
		log.Printf("exiting with code %v requested by env %s", code, exitEnvKey)
		os.Exit(code)
	}

	tmpl, err := template.New("index.html.tmpl").Funcs(template.FuncMap{"fields": fields}).ParseFiles("index.html.tmpl")
	if err != nil {
		log.Fatal(err)
	}
	tmpl.Funcs(template.FuncMap{
		"fields": fields,
	})

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {

		cpuInfo, _ := cpu.Info()
		diskInfo, _ := disk.Partitions(true)
		hostInfo, _ := host.Info()
		virtualMemory, _ := mem.VirtualMemory()
		netInfo, _ := neti.IOCounters(true)

		ip, _, err := net.SplitHostPort(req.RemoteAddr)
		if err != nil {
			fmt.Fprintf(w, "req.RemoteAddr: %s is not ip:port", req.RemoteAddr)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if realIP := req.Header.Get(realIPHeader); len(realIP) != 0 {
			ip = realIP
		}

		log.Printf("%s on %s request from %s\n", req.Method, req.URL.Path, ip)
		tmpl.Execute(w, struct {
			Text          string
			CpuInfo       []cpu.InfoStat
			DiskInfo      []disk.PartitionStat
			HostInfo      *host.InfoStat
			VirtualMemory *mem.VirtualMemoryStat
			NetInfo       []neti.IOCountersStat
		}{
			Text:          os.Getenv(responseEnvKey),
			CpuInfo:       cpuInfo,
			DiskInfo:      diskInfo,
			HostInfo:      hostInfo,
			VirtualMemory: virtualMemory,
			NetInfo:       netInfo,
		})
	})

	port := strconv.Itoa(defaultPort)
	if len(os.Getenv(portEnvKey)) != 0 {
		port = os.Getenv(portEnvKey)
	}
	addr := ":" + port

	log.Printf("starting HTTP server on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

// fields returns map of field names and values for struct s.
func fields(s interface{}) (map[string]interface{}, error) {
	v := reflect.Indirect(reflect.ValueOf(s))
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%T is not a struct", s)
	}
	m := make(map[string]interface{})
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		sv := t.Field(i)
		m[sv.Name] = v.Field(i).Interface()
	}
	return m, nil
}
