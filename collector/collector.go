package collector

import (
	"crypto/tls"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Mautu/eureka_exporter/conf"
	"github.com/prometheus/client_golang/prometheus"
	plog "github.com/prometheus/common/log"
	"go.uber.org/zap"
)

// Metrics 指标结构体
type Metrics struct {
	metrics map[string]*prometheus.Desc
	conf    *conf.Config
	mutex   sync.Mutex
}
type Port struct {
	Enabled string `xml:"enabled,attr"`
	Port    string `xml:",innerxml"`
}
type Instance struct {
	IpAddr           string `xml:"ipAddr"`
	App              string `xml:"app"`
	Status           string `xml:"status"`
	Overriddenstatus string `xml:"overriddenstatus"`
	Port             Port   `xml:"port"`
	//Portenable       PortEnable `xml:"port"`
	SecurePort Port `xml:"securePort"`
	//SecurePortEnable PortEnable `xml:"securePort"`
	CountryId string `xml:"countryId"`
}
type Application struct {
	Name      string     `xml:"name"`
	Instances []Instance `xml:"instance"`
}
type Applications struct {
	XMLName      xml.Name      `xml:"applications"`
	Applications []Application `xml:"application"`
}

/**
 * 函数：newGlobalMetric
 * 功能：创建指标描述符
 */
func newGlobalMetric(metricName string, docString string, labels []string) *prometheus.Desc {
	return prometheus.NewDesc(metricName, docString, labels, nil)
}

// Describe 功能：传递结构体中的指标描述符到channel
func (c *Metrics) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range c.metrics {
		ch <- m
	}
}

// NewMetrics 功能：初始化指标
func NewMetrics(c *conf.Config) *Metrics {
	return &Metrics{
		conf: c,
		metrics: map[string]*prometheus.Desc{
			"registry_service_status": newGlobalMetric("registry_service_status", "check eureka service registry status", []string{"registry_center", "address", "application", "port", "portenable", "secureport", "secureportenable"}),
		},
	}
}

// {'dubbo': '2.6.2', 'roles': 'provider', 'application': 'tpa-provider', 'address': '10.11.4.23', 'interface': 'com.cmiot.tpa.api.denghong.timeline.QueryDeviceRegionList', 'port': '20885'}
// Collect 功能：抓取最新的数据，传递给channel
func (c *Metrics) Collect(ch chan<- prometheus.Metric) {
	c.mutex.Lock() // 加锁
	defer c.mutex.Unlock()
	status := map[string]float64{"DOWN": 0.0, "UP": 1.0}
	applications := getEurekainfo(c.conf.Url)
	for _, application := range applications.Applications {
		for _, instance := range application.Instances {
			statu, ok := status[instance.Status]
			if !ok {
				statu = 2.0
			}
			ch <- prometheus.MustNewConstMetric(c.metrics["registry_service_status"], prometheus.GaugeValue, statu, "eureka", instance.IpAddr, instance.App, instance.Port.Port, instance.Port.Enabled, instance.SecurePort.Port, instance.SecurePort.Enabled)
		}
	}

}
func getEurekainfo(url string) Applications {
	content, _ := gethttpresponse("", url, "GET", "", "", "")
	var app Applications
	err := xml.Unmarshal(content, &app)
	if err != nil {
		plog.Fatal("error", err)

	}
	return app
}
func gethttpresponse(post string, url string, method string, user string, passwd string, auth string) ([]byte, http.Header) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{
		Timeout:   3 * time.Second,
		Transport: tr,
	}
	data := post
	req, err := http.NewRequest(method, url, strings.NewReader(data))
	if err != nil {
		plog.Errorln("request init error", zap.String("data", data), zap.String("error", err.Error()))
		return nil, nil
	}
	//	req.Header.Set("Content-Type", "application/json")
	//	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/xml")
	if user != "" && passwd != "" {
		req.SetBasicAuth(user, passwd)
	}
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	res, err := httpClient.Do(req)
	if err != nil {

		plog.Errorln("request error", zap.String("url", url), zap.String("error", err.Error()))
		return []byte("error"), nil
	}
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	// plog.Infoln(url, string(body), res.StatusCode)
	return body, res.Header
}
