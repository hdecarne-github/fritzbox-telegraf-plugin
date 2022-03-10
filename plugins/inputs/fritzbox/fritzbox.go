// Package rand is loosely based off of https://github.com/danielnelson/telegraf-plugins
package fritzbox

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
)

type FritzBox struct {
	Hosts                   []string `toml:"hosts"`
	Timeout                 int      `toml:"timeout"`
	GetStatusInfo           bool     `toml:"get_status_info"`
	GetAddonInfos           bool     `toml:"get_addon_infos"`
	GetCommonLinkProperties bool     `toml:"get_common_link_properties"`
	Debug                   bool     `toml:"debug"`

	cachedClient *http.Client
}

func NewFritzBox() *FritzBox {
	return &FritzBox{
		Hosts:                   []string{"fritz.box"},
		Timeout:                 5,
		GetStatusInfo:           true,
		GetAddonInfos:           true,
		GetCommonLinkProperties: true}
}

func (fb *FritzBox) SampleConfig() string {
	return `
  ## The fritz devices to query (host name or ip)
  # hosts = ["fritz.box"]
  ## The http timeout to use (in seconds)
  # timeout = 5
`
}

func (fb *FritzBox) Description() string {
	return "Gather FritzBox status"
}

func (fb *FritzBox) Gather(a telegraf.Accumulator) error {
	for _, baseUrl := range fb.Hosts {
		a.AddError(fb.gatherFritzBox(a, baseUrl))
	}
	return nil
}

func (fb *FritzBox) gatherFritzBox(a telegraf.Accumulator, host string) error {
	fields := make(map[string]interface{})
	tags := make(map[string]string)
	tags["host"] = host
	if fb.GetStatusInfo {
		err := fb.gatherFritzBoxStatusInfo(host, fields, tags)
		if err != nil {
			return err
		}
	}
	if fb.GetAddonInfos {
		err := fb.gatherFritzBoxAddonInfos(host, fields, tags)
		if err != nil {
			return err
		}
	}
	if fb.GetCommonLinkProperties {
		err := fb.gatherFritzBoxCommonLinkProperties(host, fields, tags)
		if err != nil {
			return err
		}
	}
	if len(fields) > 0 {
		a.AddCounter("fritzbox", fields, tags)
	}
	return nil
}

const getStatusInfoEndpoint = "http://%s:49000/igdupnp/control/WANIPConn1"
const getStatusInfoAction = "urn:schemas-upnp-org:service:WANIPConnection:1#GetStatusInfo"
const getStatusInfoRequest = `
	<?xml version="1.0" encoding="utf-8" ?>
    <s:Envelope s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/" xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
        <s:Body>
            <u:GetStatusInfo xmlns:u="urn:schemas-upnp-org:service:WANIPConnection:1" />
        </s:Body>
    </s:Envelope>`

func (fb *FritzBox) gatherFritzBoxStatusInfo(host string, fields map[string]interface{}, tags map[string]string) error {
	endpoint := fmt.Sprintf(getStatusInfoEndpoint, host)
	response, err := fb.soapAction(endpoint, getStatusInfoAction, getStatusInfoRequest)
	if err != nil {
		return err
	}
	statusInfo := struct {
		NewConnectionStatus    string `xml:"Body>GetStatusInfoResponse>NewConnectionStatus"`
		NewLastConnectionError string `xml:"Body>GetStatusInfoResponse>NewLastConnectionError"`
		NewUptime              string `xml:"Body>GetStatusInfoResponse>NewUptime"`
	}{}
	if response != nil {
		err := xml.Unmarshal(response, &statusInfo)
		if err != nil {
			return err
		}
		fields["uptime"] = statusInfo.NewUptime
	}
	return nil
}

const getAddonInfosEndpoint = "http://%s:49000/igdupnp/control/WANCommonIFC1"
const getAddonInfosAction = "urn:schemas-upnp-org:service:WANCommonInterfaceConfig:1#GetAddonInfos"
const getAddonInfosRequest = `
	<?xml version="1.0" encoding="utf-8" ?>
    <s:Envelope s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/" xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
        <s:Body>
            <u:GetCommonLinkProperties xmlns:u="urn:schemas-upnp-org:service:WANCommonInterfaceConfig:1" />
        </s:Body>
    </s:Envelope>`

func (fb *FritzBox) gatherFritzBoxAddonInfos(host string, fields map[string]interface{}, tags map[string]string) error {
	endpoint := fmt.Sprintf(getAddonInfosEndpoint, host)
	response, err := fb.soapAction(endpoint, getAddonInfosAction, getAddonInfosRequest)
	if err != nil {
		return err
	}
	addonInfos := struct {
		NewByteSendRate       string `xml:"Body>GetAddonInfosResponse>NewByteSendRate"`
		NewByteReceiveRate    string `xml:"Body>GetAddonInfosResponse>NewByteReceiveRate"`
		NewTotalBytesSent     string `xml:"Body>GetAddonInfosResponse>NewTotalBytesSent"`
		NewTotalBytesReceived string `xml:"Body>GetAddonInfosResponse>NewTotalBytesReceived"`
		TotalBytesSent64      string `xml:"Body>GetAddonInfosResponse>NewX_AVM_DE_TotalBytesSent64"`
		TotalBytesReceived64  string `xml:"Body>GetAddonInfosResponse>NewX_AVM_DE_TotalBytesReceived64"`
	}{}
	if response != nil {
		err := xml.Unmarshal(response, &addonInfos)
		if err != nil {
			return err
		}
		fields["byte_send_rate"] = addonInfos.NewByteSendRate
		fields["byte_receive_rate"] = addonInfos.NewByteReceiveRate
		fields["total_bytes_sent"] = addonInfos.NewTotalBytesSent
		fields["total_bytes_received"] = addonInfos.NewTotalBytesReceived
		fields["total_bytes_sent64"] = addonInfos.TotalBytesSent64
		fields["total_bytes_received64"] = addonInfos.TotalBytesReceived64
	}
	return nil
}

const getCommonLinkPropertiesEndpoint = "http://%s:49000/igdupnp/control/WANCommonIFC1"
const getCommonLinkPropertiesAction = "urn:schemas-upnp-org:service:WANCommonInterfaceConfig:1#GetCommonLinkProperties"
const getCommonLinkPropertiesRequest = `
	<?xml version="1.0" encoding="utf-8" ?>
		<s:Envelope s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/" xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
			<s:Body>
            	<u:GetCommonLinkProperties xmlns:u="urn:schemas-upnp-org:service:WANCommonInterfaceConfig:1" />
	        </s:Body>
    </s:Envelope>`

func (fb *FritzBox) gatherFritzBoxCommonLinkProperties(host string, fields map[string]interface{}, tags map[string]string) error {
	endpoint := fmt.Sprintf(getCommonLinkPropertiesEndpoint, host)
	response, err := fb.soapAction(endpoint, getCommonLinkPropertiesAction, getCommonLinkPropertiesRequest)
	if err != nil {
		return err
	}
	commonLinkProperties := struct {
		NewLayer1UpstreamMaxBitRate   string `xml:"Body>GetCommonLinkPropertiesResponse>NewLayer1UpstreamMaxBitRate"`
		NewLayer1DownstreamMaxBitRate string `xml:"Body>GetCommonLinkPropertiesResponse>NewLayer1DownstreamMaxBitRate"`
	}{}
	if response != nil {
		err := xml.Unmarshal(response, &commonLinkProperties)
		if err != nil {
			return err
		}
		fields["upstream_bit_rate"] = commonLinkProperties.NewLayer1UpstreamMaxBitRate
		fields["downstream_bit_rate"] = commonLinkProperties.NewLayer1DownstreamMaxBitRate
	}
	return nil
}

func (fb *FritzBox) soapAction(endpoint string, action string, request string) ([]byte, error) {
	if fb.Debug {
		log.Printf("Invoking action %s on endpoint %s ...", action, endpoint)
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(request))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "text/xml")
	req.Header.Add("SoapAction", action)
	client := fb.getClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if fb.Debug {
		log.Printf("Status code: %d", resp.StatusCode)
	}
	if resp.StatusCode != 200 {
		return nil, nil
	}
	defer resp.Body.Close()
	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if fb.Debug {
		log.Printf("Response: %s", response)
	}
	return response, nil
}

func (fb *FritzBox) getClient() *http.Client {
	if fb.cachedClient == nil {
		transport := &http.Transport{
			ResponseHeaderTimeout: time.Duration(fb.Timeout) * time.Second,
		}
		fb.cachedClient = &http.Client{
			Transport: transport,
			Timeout:   time.Duration(fb.Timeout) * time.Second,
		}
	}
	return fb.cachedClient
}

func init() {
	inputs.Add("fritzbox", func() telegraf.Input {
		return NewFritzBox()
	})
}
