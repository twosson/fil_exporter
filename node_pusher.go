package main

import (
    "fmt"
    "net/http"
    _ "net/http/pprof"
    "os"
    "errors"
    "bytes"
    "time"
    "sort"
    "net"
    "io/ioutil"
    "os/signal"


    "github.com/prometheus/common/expfmt"

    "github.com/prometheus/common/promlog"
    "github.com/prometheus/common/promlog/flag"

    "github.com/go-kit/kit/log"
    "github.com/go-kit/kit/log/level"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/common/version"
    "github.com/prometheus/node_exporter/collector"
    kingpin "gopkg.in/alecthomas/kingpin.v2"
)



func start(l log.Logger) (prometheus.Gatherer, error) {
    nc, err := collector.NewNodeCollector(l)
    if err != nil {
        return nil, fmt.Errorf("couldn't create collector: %s", err)
    }

    // Only log the creation of an unfiltered handler, which should happen
    // only once upon startup.
    // level.Info(logger).Log("msg", "Enabled collectors")

    collectors := []string{}
    for n := range nc.Collectors {
        collectors = append(collectors, n)
    }

    sort.Strings(collectors)
    for _, c := range collectors {
        level.Info(l).Log("collector", c)
    }

    r := prometheus.NewRegistry()
    r.MustRegister(version.NewCollector("node_exporter"))
    if err := r.Register(nc); err != nil {
        return nil, fmt.Errorf("couldn't register node collector: %s", err)
    }

    gather := prometheus.Gatherer(r)

    return gather, nil
}

// 获取内网ip地址
func getLocalIP() (string, error) {
    addrs, err := net.InterfaceAddrs()
    if err != nil {
        return "", err
    }

    for _, address := range addrs {
        // 检查ip地址判断是否回环地址
        if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
            if ipnet.IP.To4() != nil {
                return ipnet.IP.String(), nil
            }
        }
    }

    return "", errors.New("no interface found")
}

func main() {
    var (
        disableDefaultCollectors = kingpin.Flag(
            "collector.disable-defaults",
            "Set all collectors to disabled by default.",
        ).Default("false").Bool()
        pusherToken = kingpin.Flag(
            "pusher.token",
            "Token for push gateway",
        ).Default("").String()
        gatewayAddress = kingpin.Flag(
            "pusher.address",
            "Address of push gateway",
        ).Default("http://129.204.3.3:32583").String()
    )

    promlogConfig := &promlog.Config{}
    flag.AddFlags(kingpin.CommandLine, promlogConfig)
    kingpin.Version(version.Print("node_exporter"))
    kingpin.HelpFlag.Short('h')
    kingpin.Parse()
    logger := promlog.New(promlogConfig)

    if *disableDefaultCollectors {
        collector.DisableDefaultCollectors()
    }
    level.Info(logger).Log("msg", "Starting node_exporter", "version", version.Info())
    level.Info(logger).Log("msg", "Build context", "build_context", version.BuildContext())

    gather, err := start(logger);
    if err != nil {
        panic(err)
    }

    shutdowchan := make(chan os.Signal, 1)
    signal.Notify(shutdowchan, os.Interrupt)


    go func() {
        for {
            <- time.After(time.Second*10)

            mfs, err := gather.Gather()
            if err != nil {
                panic(err)
            }

            contentType := expfmt.FmtText

            
            var b bytes.Buffer

            enc := expfmt.NewEncoder(&b, contentType)

            for _, mf := range mfs {
                if err := enc.Encode(mf); err != nil {
                    panic(err)
                }
            }

            level.Info(logger).Log("msg", "start http post")
            clt := http.Client{}
            ip, err := getLocalIP() 
            if err != nil {
                level.Warn(logger).Log("get ip error", err)
                panic(err)
            }

            reqest, err := http.NewRequest("POST", *gatewayAddress + "/metrics/job/node/instance/" + ip, &b)
            // reqest, err := http.NewRequest("POST", *gatewayAddress + "/metrics/job/node/instance/" + ip, &b)
            
            // level.Info(logger).Log("data", b.String())
            if err != nil {
                level.Warn(logger).Log("msg", err)
                continue
            }   

            //增加header选项
            reqest.Header.Add("Blade-Auth", "bearer " + *pusherToken)
            reqest.Header.Add("User-Agent", "node pusher")
            reqest.Header.Add("Content-Type", "text/plain;charset=utf-8")
            
            resp, err := clt.Do(reqest)
            defer resp.Body.Close()
            if err != nil {
                level.Warn(logger).Log("msg", err)
                continue
            }
            if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
                level.Debug(logger).Log("msg", "Status post success!")
            } else {
                body, err := ioutil.ReadAll(resp.Body)
                if err != nil {
                    level.Warn(logger).Log("msg", err)
                    continue
                }
                level.Info(logger).Log("body", body)
            }

            if closer, ok := enc.(expfmt.Closer); ok {
                // This in particular takes care of the final "# EOF\n" line for OpenMetrics.
                if err := closer.Close(); err != nil {
                    panic(err)
                }
            }
        }
    }()

    select {
    case <-shutdowchan:
        level.Info(logger).Log("msg", "user shutdown")
    }

}