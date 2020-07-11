package collector

import (
    "fmt"
    "strconv"
    "sync"
    "encoding/csv"
    "os/exec"
    "bytes"

    "github.com/go-kit/kit/log"
    "github.com/go-kit/kit/log/level"
    "github.com/prometheus/client_golang/prometheus"
    "gopkg.in/alecthomas/kingpin.v2"
)

// NvidiaCollector nvidia gpu collector
type nvidiaCollector struct {
    gpuStatusMutex sync.Mutex
    numDevices  *prometheus.Desc
    gpuInfo  *prometheus.Desc
    logger      log.Logger
}

const (
    gpuInfoSubsystem = "gpu"
)

var (
    enableGPUInfo = kingpin.Flag("collector.gpu.info", "Enables metric gpu info").Default("true").Bool()
)

func init() {
    registerCollector("gpu", defaultEnabled, NewNvidiaCollector)
}

func NewNvidiaCollector(logger log.Logger) (Collector, error) {
    namespace := "nvidia"
    nc := &nvidiaCollector{
        numDevices: prometheus.NewDesc(
            prometheus.BuildFQName(namespace, gpuInfoSubsystem, "num_devices"),
            "Number of Nvidia GPU devices",
            nil,
            nil,
        ),

        gpuInfo: prometheus.NewDesc(
            prometheus.BuildFQName(namespace, gpuInfoSubsystem, "info"),
            "GPU information",
            []string{"gpu", "metrics"}, nil,
        ),

        logger: logger,
    }
    return nc, nil
}

func (nc *nvidiaCollector)updateMetrics(ch chan<- prometheus.Metric, gpuNum string, name string, value string) {
    v, err := strconv.ParseFloat(value, 64)
    if err != nil {
        return
    }

    ch <- prometheus.MustNewConstMetric(
        nc.gpuInfo, prometheus.GaugeValue, v, gpuNum, name,
    )
}
// Collect implement prometheus Collector interface
func (nc *nvidiaCollector) Update(ch chan<- prometheus.Metric) error {
    if *enableGPUInfo {
        level.Debug(nc.logger).Log()
        out, err := exec.Command(
            "nvidia-smi",
            "--query-gpu=index,name,temperature.gpu,utilization.gpu,utilization.memory,memory.total,memory.free,memory.used",
            "--format=csv,noheader,nounits",
        ).Output()

        if err != nil {
            return fmt.Errorf("couldn't get gpu info: %w", err)
        }

        csvReader := csv.NewReader(bytes.NewReader(out))
        csvReader.TrimLeadingSpace = true
        records, err := csvReader.ReadAll()
        if err != nil {
            return fmt.Errorf("couldn't parse gpu info: %w", err)
        }

        level.Debug(nc.logger).Log("gpu", "gpu info collected")

        count := len(records)
        ch <- prometheus.MustNewConstMetric(
            nc.numDevices,
            prometheus.CounterValue,
            float64(count),
        )

        names := []string{"name","temperature.gpu","utilization.gpu","utilization.memory","memory.total","memory.free","memory.used"}
        for i, infoPerCore := range records {
            gpuNum := strconv.Itoa(i)

            for j, val := range infoPerCore[1:] {
                nc.updateMetrics(ch, gpuNum, names[j], val)
            }
        }
    }

    return nil
}