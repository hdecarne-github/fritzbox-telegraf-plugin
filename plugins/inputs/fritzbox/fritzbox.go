package fritzbox

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
)

type deviceInfo struct {
	BaseUrl              *url.URL
	Login                string
	Password             string
	ServiceInfo          *tr64Desc
	cachedAuthentication [2]string
}

type tr64Desc struct {
	FriendlyName string                  `xml:"device>friendlyName"`
	Services     []tr64DescDeviceService `xml:"device>serviceList>service"`
	Devices      []tr64DescDevice        `xml:"device>deviceList>device"`
}

type tr64DescDevice struct {
	FriendlyName string                  `xml:"friendlyName"`
	Services     []tr64DescDeviceService `xml:"serviceList>service"`
	Devices      []tr64DescDevice        `xml:"deviceList>device"`
}

type tr64DescDeviceService struct {
	ServiceType string `xml:"serviceType"`
	ServiceId   string `xml:"serviceId"`
	ControlURL  string `xml:"controlURL"`
}

func (s *tr64DescDeviceService) ShortServiceId() string {
	split := strings.Split(s.ServiceId, ":")
	return split[len(split)-1]
}

type FritzBox struct {
	Devices     [][]string `toml:"devices"`
	Timeout     int        `toml:"timeout"`
	GetWLANInfo bool       `toml:"get_wlan_info"`
	GetWANInfo  bool       `toml:"get_wan_info"`
	GetDSLInfo  bool       `toml:"get_dsl_info"`
	GetPPPInfo  bool       `toml:"get_ppp_info"`
	Debug       bool       `toml:"debug"`

	deviceInfos  map[string]*deviceInfo
	cachedClient *http.Client
}

func NewFritzBox() *FritzBox {
	return &FritzBox{
		Devices:     [][]string{{"fritz.box", "", ""}},
		Timeout:     5,
		GetWLANInfo: true,
		GetWANInfo:  true,
		GetDSLInfo:  true,
		GetPPPInfo:  true,

		deviceInfos: make(map[string]*deviceInfo)}
}

func (fb *FritzBox) SampleConfig() string {
	return `
  ## The fritz devices to query (multiple triples of base url, login, password)
  # devices = [["http://fritz.box:49000", "", ""]]
  ## The http timeout to use (in seconds)
  # timeout = 5
  ## Process WLAN services (if found)
  # get_wlan_info = true
  ## Process WAN services (if found)
  # get_wan_info = true
  ## Process DSL services (if found)
  # get_dsl_info = true
  ## Process PPP services (if found)
  # get_dsl_info = true
  ## Enable debug output
  # debug = false
`
}

func (fb *FritzBox) Description() string {
	return "Gather FritzBox stats"
}

func (fb *FritzBox) Gather(a telegraf.Accumulator) error {
	if len(fb.Devices) == 0 {
		return errors.New("fritzbox: Empty device list")
	}
	for _, device := range fb.Devices {
		if len(device) != 3 {
			return fmt.Errorf("fritzbox: Invalid device entry: %s", device)
		}
		rawBaseUrl := device[0]
		login := device[1]
		password := device[2]
		deviceInfo, err := fb.fetchDeviceInfo(rawBaseUrl, login, password)
		if err == nil {
			a.AddError(fb.processRootDevice(a, deviceInfo))
		} else {
			a.AddError(err)
		}
	}
	return nil
}

func (fb *FritzBox) processRootDevice(a telegraf.Accumulator, deviceInfo *deviceInfo) error {
	if fb.Debug {
		log.Printf("Considering root device: %s", deviceInfo.ServiceInfo.FriendlyName)
	}
	fb.processServices(a, deviceInfo, deviceInfo.ServiceInfo.Services)
	fb.processDevices(a, deviceInfo, deviceInfo.ServiceInfo.Devices)
	return nil
}

func (fb *FritzBox) processDevices(a telegraf.Accumulator, deviceInfo *deviceInfo, devices []tr64DescDevice) error {
	for _, device := range devices {
		if fb.Debug {
			log.Printf("Considering device: %s", device.FriendlyName)
		}
		fb.processServices(a, deviceInfo, device.Services)
		fb.processDevices(a, deviceInfo, device.Devices)
	}
	return nil
}

func (fb *FritzBox) processServices(a telegraf.Accumulator, deviceInfo *deviceInfo, services []tr64DescDeviceService) error {
	for _, service := range services {
		if fb.Debug {
			log.Printf("Considering service type: %s", service.ServiceType)
		}
		if strings.HasPrefix(service.ServiceType, "urn:dslforum-org:service:WLANConfiguration:") {
			if fb.GetWLANInfo {
				a.AddError(fb.processWLANConfigurationService(a, deviceInfo, &service))
			}
		} else if strings.HasPrefix(service.ServiceType, "urn:dslforum-org:service:WANCommonInterfaceConfig:") {
			if fb.GetWANInfo {
				a.AddError(fb.processWANCommonInterfaceConfigService(a, deviceInfo, &service))
			}
		} else if strings.HasPrefix(service.ServiceType, "urn:dslforum-org:service:WANDSLInterfaceConfig:") {
			if fb.GetDSLInfo {
				a.AddError(fb.processDSLInterfaceConfigService(a, deviceInfo, &service))
			}
		} else if strings.HasPrefix(service.ServiceType, "urn:dslforum-org:service:WANPPPConnection:") {
			if fb.GetPPPInfo {
				a.AddError(fb.processPPPConnectionService(a, deviceInfo, &service))
			}
		}

	}
	return nil
}

func (fb *FritzBox) processWLANConfigurationService(a telegraf.Accumulator, deviceInfo *deviceInfo, service *tr64DescDeviceService) error {
	info := struct {
		Status  string `xml:"Body>GetInfoResponse>NewStatus"`
		Channel string `xml:"Body>GetInfoResponse>NewChannel"`
		SSID    string `xml:"Body>GetInfoResponse>NewSSID"`
	}{}
	err := fb.invokeDeviceService(deviceInfo, service, "GetInfo", &info)
	if err != nil {
		return err
	}
	totalAssociations := struct {
		TotalAssociations int `xml:"Body>GetTotalAssociationsResponse>NewTotalAssociations"`
	}{}
	err = fb.invokeDeviceService(deviceInfo, service, "GetTotalAssociations", &totalAssociations)
	if err != nil {
		return err
	}
	connectionInfo := struct {
		SSID           string `xml:"Body>X_AVM-DE_GetWLANConnectionInfoResponse>NewSSID"`
		Channel        string `xml:"Body>X_AVM-DE_GetWLANConnectionInfoResponse>NewChannel"`
		SignalStrength int    `xml:"Body>X_AVM-DE_GetWLANConnectionInfoResponse>NewX_AVM-DE_SignalStrength"`
		Speed          int    `xml:"Body>X_AVM-DE_GetWLANConnectionInfoResponse>NewX_AVM-DE_Speed"`
		SpeedRX        int    `xml:"Body>X_AVM-DE_GetWLANConnectionInfoResponse>NewX_AVM-DE_SpeedRX"`
		SpeedMax       int    `xml:"Body>X_AVM-DE_GetWLANConnectionInfoResponse>NewX_AVM-DE_SpeedMax"`
		SpeedRXMax     int    `xml:"Body>X_AVM-DE_GetWLANConnectionInfoResponse>NewX_AVM-DE_SpeedRXMax"`
	}{}
	connectionInfoErr := fb.invokeDeviceService(deviceInfo, service, "X_AVM-DE_GetWLANConnectionInfo", &connectionInfo)
	if info.Status == "Up" {
		tags := make(map[string]string)
		tags["fritz_device"] = deviceInfo.BaseUrl.Hostname()
		tags["service"] = service.ShortServiceId()
		tags["access_point"] = deviceInfo.BaseUrl.Hostname() + ":" + info.SSID + ":" + info.Channel
		fields := make(map[string]interface{})
		fields["total_associations"] = totalAssociations.TotalAssociations
		if connectionInfoErr == nil && connectionInfo.SSID == info.SSID && connectionInfo.Channel == info.Channel {
			fields["bridge_signal_strength"] = connectionInfo.SignalStrength
			fields["bridge_speed"] = connectionInfo.Speed
			fields["bridge_speed_rx"] = connectionInfo.SpeedRX
			fields["bridge_speed_max"] = connectionInfo.SpeedMax
			fields["bridge_speed_rx_max"] = connectionInfo.SpeedRXMax
		} else {
			fields["bridge_signal_strength"] = 0
			fields["bridge_speed"] = 0
			fields["bridge_speed_rx"] = 0
			fields["bridge_speed_max"] = 0
			fields["bridge_speed_rx_max"] = 0
		}
		a.AddCounter("fritzbox_wlan", fields, tags)
	}
	return nil
}

func (fb *FritzBox) processWANCommonInterfaceConfigService(a telegraf.Accumulator, deviceInfo *deviceInfo, service *tr64DescDeviceService) error {
	commonLinkProperties := struct {
		Layer1UpstreamMaxBitRate   int    `xml:"Body>GetCommonLinkPropertiesResponse>NewLayer1UpstreamMaxBitRate"`
		Layer1DownstreamMaxBitRate int    `xml:"Body>GetCommonLinkPropertiesResponse>NewLayer1DownstreamMaxBitRate"`
		PhysicalLinkStatus         string `xml:"Body>GetCommonLinkPropertiesResponse>NewPhysicalLinkStatus"`
		UpstreamCurrentMaxSpeed    int    `xml:"Body>GetCommonLinkPropertiesResponse>NewX_AVM-DE_UpstreamCurrentMaxSpeed"`
		DownstreamCurrentMaxSpeed  int    `xml:"Body>GetCommonLinkPropertiesResponse>NewX_AVM-DE_DownstreamCurrentMaxSpeed"`
	}{}
	err := fb.invokeDeviceService(deviceInfo, service, "GetCommonLinkProperties", &commonLinkProperties)
	if err != nil {
		return err
	}
	totalBytesSent := struct {
		TotalBytesSent int `xml:"Body>GetTotalBytesSentResponse>NewTotalBytesSent"`
	}{}
	err = fb.invokeDeviceService(deviceInfo, service, "GetTotalBytesSent", &totalBytesSent)
	if err != nil {
		return err
	}
	totalBytesReceived := struct {
		TotalBytesReceived int `xml:"Body>GetTotalBytesReceivedResponse>NewTotalBytesReceived"`
	}{}
	err = fb.invokeDeviceService(deviceInfo, service, "GetTotalBytesReceived", &totalBytesReceived)
	if err != nil {
		return err
	}
	if commonLinkProperties.PhysicalLinkStatus == "Up" {
		tags := make(map[string]string)
		tags["fritz_device"] = deviceInfo.BaseUrl.Hostname()
		tags["service"] = service.ShortServiceId()
		fields := make(map[string]interface{})
		fields["layer1_upstream_max_bit_rate"] = commonLinkProperties.Layer1UpstreamMaxBitRate
		fields["layer1_downstream_max_bit_rate"] = commonLinkProperties.Layer1DownstreamMaxBitRate
		fields["upstream_current_max_speed"] = commonLinkProperties.UpstreamCurrentMaxSpeed
		fields["downstream_current_max_speed"] = commonLinkProperties.DownstreamCurrentMaxSpeed
		fields["total_bytes_sent"] = totalBytesSent.TotalBytesSent
		fields["total_bytes_received"] = totalBytesReceived.TotalBytesReceived
		a.AddCounter("fritzbox_wan", fields, tags)
	}
	return nil
}

func (fb *FritzBox) processDSLInterfaceConfigService(a telegraf.Accumulator, deviceInfo *deviceInfo, service *tr64DescDeviceService) error {
	info := struct {
		Status                string `xml:"Body>GetInfoResponse>NewStatus"`
		UpstreamCurrRate      int    `xml:"Body>GetInfoResponse>NewUpstreamCurrRate"`
		DownstreamCurrRate    int    `xml:"Body>GetInfoResponse>NewDownstreamCurrRate"`
		UpstreamMaxRate       int    `xml:"Body>GetInfoResponse>NewUpstreamMaxRate"`
		DownstreamMaxRate     int    `xml:"Body>GetInfoResponse>NewDownstreamMaxRate"`
		UpstreamNoiseMargin   int    `xml:"Body>GetInfoResponse>NewUpstreamNoiseMargin"`
		DownstreamNoiseMargin int    `xml:"Body>GetInfoResponse>NewDownstreamNoiseMargin"`
		UpstreamAttenuation   int    `xml:"Body>GetInfoResponse>NewUpstreamAttenuation"`
		DownstreamAttenuation int    `xml:"Body>GetInfoResponse>NewDownstreamAttenuation"`
		UpstreamPower         int    `xml:"Body>GetInfoResponse>NewUpstreamPower"`
		DownstreamPower       int    `xml:"Body>GetInfoResponse>NewDownstreamPower"`
	}{}
	err := fb.invokeDeviceService(deviceInfo, service, "GetInfo", &info)
	if err != nil {
		return err
	}
	statisticsTotal := struct {
		ReceiveBlocks       int `xml:"Body>GetStatisticsTotalResponse>NewReceiveBlocks"`
		TransmitBlocks      int `xml:"Body>GetStatisticsTotalResponse>NewTransmitBlocks"`
		CellDelin           int `xml:"Body>GetStatisticsTotalResponse>NewCellDelin"`
		LinkRetrain         int `xml:"Body>GetStatisticsTotalResponse>NewLinkRetrain"`
		InitErrors          int `xml:"Body>GetStatisticsTotalResponse>NewInitErrors"`
		InitTimeouts        int `xml:"Body>GetStatisticsTotalResponse>NewInitTimeouts"`
		LossOfFraming       int `xml:"Body>GetStatisticsTotalResponse>NewLossOfFraming"`
		ErroredSecs         int `xml:"Body>GetStatisticsTotalResponse>NewErroredSecs"`
		SeverelyErroredSecs int `xml:"Body>GetStatisticsTotalResponse>NewSeverelyErroredSecs"`
		FECErrors           int `xml:"Body>GetStatisticsTotalResponse>NewFECErrors"`
		ATUCFECErrors       int `xml:"Body>GetStatisticsTotalResponse>NewATUCFECErrors"`
		HECErrors           int `xml:"Body>GetStatisticsTotalResponse>NewHECErrors"`
		ATUCHECErrors       int `xml:"Body>GetStatisticsTotalResponse>NewATUCHECErrors"`
		CRCErrors           int `xml:"Body>GetStatisticsTotalResponse>NewCRCErrors"`
		ATUCCRCErrors       int `xml:"Body>GetStatisticsTotalResponse>NewATUCCRCErrors"`
	}{}
	err = fb.invokeDeviceService(deviceInfo, service, "GetStatisticsTotal", &statisticsTotal)
	if err != nil {
		return err
	}
	if info.Status == "Up" {
		tags := make(map[string]string)
		tags["fritz_device"] = deviceInfo.BaseUrl.Hostname()
		tags["service"] = service.ShortServiceId()
		fields := make(map[string]interface{})
		fields["upstream_curr_rate"] = info.UpstreamCurrRate
		fields["downstream_curr_rate"] = info.DownstreamCurrRate
		fields["upstream_max_rate"] = info.UpstreamMaxRate
		fields["downstream_max_rate"] = info.DownstreamMaxRate
		fields["upstream_noise_margin"] = info.UpstreamNoiseMargin
		fields["downstream_noise_margin"] = info.DownstreamNoiseMargin
		fields["upstream_attenuation"] = info.UpstreamAttenuation
		fields["downstream_attenuation"] = info.DownstreamAttenuation
		fields["upstream_power"] = info.UpstreamPower
		fields["downstream_power"] = info.DownstreamPower
		fields["receive_blocks"] = statisticsTotal.ReceiveBlocks
		fields["transmit_blocks"] = statisticsTotal.TransmitBlocks
		fields["cell_delin"] = statisticsTotal.CellDelin
		fields["link_retrain"] = statisticsTotal.LinkRetrain
		fields["init_errors"] = statisticsTotal.InitErrors
		fields["init_timeouts"] = statisticsTotal.InitTimeouts
		fields["loss_of_framing"] = statisticsTotal.LossOfFraming
		fields["errored_secs"] = statisticsTotal.ErroredSecs
		fields["severly_errored_secs"] = statisticsTotal.SeverelyErroredSecs
		fields["fec_errors"] = statisticsTotal.FECErrors
		fields["atuc_fec_errors"] = statisticsTotal.ATUCFECErrors
		fields["hec_errors"] = statisticsTotal.HECErrors
		fields["atuc_hec_errors"] = statisticsTotal.ATUCHECErrors
		fields["crc_errors"] = statisticsTotal.CRCErrors
		fields["atuc_crc_errors"] = statisticsTotal.ATUCCRCErrors
		a.AddCounter("fritzbox_dsl", fields, tags)
	}
	return nil
}

func (fb *FritzBox) processPPPConnectionService(a telegraf.Accumulator, deviceInfo *deviceInfo, service *tr64DescDeviceService) error {
	info := struct {
		ConnectionStatus     string `xml:"Body>GetInfoResponse>NewConnectionStatus"`
		Uptime               int    `xml:"Body>GetInfoResponse>NewUptime"`
		UpstreamMaxBitRate   int    `xml:"Body>GetInfoResponse>NewUpstreamMaxBitRate"`
		DownstreamMaxBitRate int    `xml:"Body>GetInfoResponse>NewDownstreamMaxBitRate"`
	}{}
	err := fb.invokeDeviceService(deviceInfo, service, "GetInfo", &info)
	if err != nil {
		return err
	}
	if info.ConnectionStatus == "Connected" {
		tags := make(map[string]string)
		tags["fritz_device"] = deviceInfo.BaseUrl.Hostname()
		tags["service"] = service.ShortServiceId()
		fields := make(map[string]interface{})
		fields["uptime"] = info.Uptime
		fields["upstream_max_bit_rate"] = info.UpstreamMaxBitRate
		fields["downstream_max_bit_rate"] = info.DownstreamMaxBitRate
		a.AddCounter("fritzbox_ppp", fields, tags)
	}
	return nil
}

func (fb *FritzBox) invokeDeviceService(deviceInfo *deviceInfo, service *tr64DescDeviceService, action string, out interface{}) error {
	controlUrl, err := url.Parse(service.ControlURL)
	if err != nil {
		return err
	}
	endpoint := deviceInfo.BaseUrl.ResolveReference(controlUrl).String()
	soapAction := fmt.Sprintf("%s#%s", service.ServiceType, action)
	requestBody := fmt.Sprintf(`
		<?xml version="1.0" encoding="utf-8" ?>
		<s:Envelope s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/" xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
			<s:Body>
				<u:%s xmlns:u="%s" />
			</s:Body>
		</s:Envelope>`, action, service.ServiceId)
	cachedAuthentication := fb.getCachedDigestAuthentication(deviceInfo, service.ServiceType)
	response, err := fb.postSoapActionRequest(endpoint, soapAction, requestBody, cachedAuthentication)
	if err != nil {
		return err
	}
	if response.StatusCode == http.StatusUnauthorized {
		authentication, err := fb.getDigestAuthentication(response, deviceInfo, service.ServiceType)
		if err == nil {
			response, err = fb.postSoapActionRequest(endpoint, soapAction, requestBody, authentication)
			if err != nil {
				return err
			}
		}
	}
	if response.StatusCode != http.StatusOK {
		return nil
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if fb.Debug {
		log.Printf("Response:\n%s", responseBody)
	}
	err = xml.Unmarshal(responseBody, out)
	if err != nil {
		return err
	}
	return nil
}

func (fb *FritzBox) postSoapActionRequest(endpoint string, action string, requestBody string, authentication string) (*http.Response, error) {
	if fb.Debug {
		log.Printf("Invoking SOAP action %s on endpoint %s ...", action, endpoint)
	}
	request, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(requestBody))
	if err != nil {
		return nil, err
	}
	request.Header.Add("Content-Type", "text/xml")
	request.Header.Add("SoapAction", action)
	if authentication != "" {
		request.Header.Add("Authorization", authentication)
	}
	client := fb.getClient()
	response, err := client.Do(request)
	if err != nil {
		return response, err
	}
	if fb.Debug {
		log.Printf("Status code: %d", response.StatusCode)
	}
	return response, nil
}

func (fb *FritzBox) getCachedDigestAuthentication(deviceInfo *deviceInfo, uri string) string {
	if deviceInfo.cachedAuthentication[0] == uri {
		return deviceInfo.cachedAuthentication[1]
	}
	return ""
}

func (fb *FritzBox) getDigestAuthentication(challenge *http.Response, deviceInfo *deviceInfo, uri string) (string, error) {
	challengeHeader := challenge.Header["Www-Authenticate"]
	if len(challengeHeader) != 1 {
		return "", errors.New("missing or unexpected WWW-Authenticate header in response")
	}
	challengeValues := make(map[string]string)
	for _, challengeHeaderValue := range strings.Split(challengeHeader[0], ",") {
		splitChallengeHeaderValue := strings.Split(challengeHeaderValue, "=")
		if len(splitChallengeHeaderValue) == 2 {
			key := splitChallengeHeaderValue[0]
			value := splitChallengeHeaderValue[1]
			if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
				value = value[1 : len(value)-1]
			}
			challengeValues[key] = value
		}
	}
	digestRealm := challengeValues["Digest realm"]
	ha1 := md5Hash(fmt.Sprintf("%s:%s:%s", deviceInfo.Login, digestRealm, deviceInfo.Password))
	ha2 := md5Hash(fmt.Sprintf("%s:%s", http.MethodPost, uri))
	nonce := challengeValues["nonce"]
	qop := challengeValues["qop"]
	cnonce := generateCNonce()
	nc := "1"
	response := md5Hash(fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, nonce, nc, cnonce, qop, ha2))
	authentication := fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", cnonce="%s", nc="%v", qop="%s", response="%s"`,
		deviceInfo.Login, digestRealm, nonce, uri, cnonce, nc, qop, response)
	deviceInfo.cachedAuthentication[0] = uri
	deviceInfo.cachedAuthentication[1] = authentication
	return authentication, nil
}

func md5Hash(in string) string {
	hash := md5.New()
	_, err := hash.Write([]byte(in))
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func generateCNonce() string {
	cnonceBytes := make([]byte, 8)
	_, err := io.ReadFull(rand.Reader, cnonceBytes)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%016x", cnonceBytes)
}

func (fb *FritzBox) fetchDeviceInfo(rawBaseUrl string, login string, password string) (*deviceInfo, error) {
	cachedDeviceInfo, cached := fb.deviceInfos[rawBaseUrl]
	if !cached {
		if fb.Debug {
			log.Printf("Querying device info for: %s", rawBaseUrl)
		}
		baseUrl, err := url.Parse(rawBaseUrl)
		if err != nil {
			return nil, err
		}

		var serviceInfo tr64Desc

		_, err = fb.fetchServiceInfo(baseUrl, "/tr64desc.xml", &serviceInfo)
		if err != nil {
			return nil, err
		}
		cachedDeviceInfo = &deviceInfo{
			BaseUrl:     baseUrl,
			Login:       login,
			Password:    password,
			ServiceInfo: &serviceInfo}
		fb.deviceInfos[rawBaseUrl] = cachedDeviceInfo
	}
	return cachedDeviceInfo, nil
}

func (fb *FritzBox) fetchServiceInfo(baseUrl *url.URL, path string, info interface{}) (*url.URL, error) {
	pathUrl, _ := url.Parse(path)
	xmlUrl := baseUrl.ResolveReference(pathUrl)
	if fb.Debug {
		log.Printf("Fetching service info from: %s", xmlUrl)
	}
	client := fb.getClient()
	resp, err := client.Get(xmlUrl.String())
	if err != nil {
		return xmlUrl, err
	}
	defer resp.Body.Close()
	return xmlUrl, xml.NewDecoder(resp.Body).Decode(info)
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
