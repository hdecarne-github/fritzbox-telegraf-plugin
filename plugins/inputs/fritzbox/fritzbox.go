// fritzbox.go
//
// Copyright (C) 2022-2023 Holger de Carne
//
// This software may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.

package fritzbox

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
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
	GetMeshInfo          bool
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
	Devices        [][]string `toml:"devices"`
	Timeout        int        `toml:"timeout"`
	TLSSkipVerify  bool       `toml:"tls_skip_verify"`
	GetDeviceInfo  bool       `toml:"get_device_info"`
	GetWLANInfo    bool       `toml:"get_wlan_info"`
	GetWANInfo     bool       `toml:"get_wan_info"`
	GetDSLInfo     bool       `toml:"get_dsl_info"`
	GetPPPInfo     bool       `toml:"get_ppp_info"`
	GetMeshInfo    []string   `toml:"get_mesh_info"`
	GetMeshClients bool       `toml:"get_mesh_clients"`
	FullQueryCycle int        `toml:"full_query_cycle"`
	Debug          bool       `toml:"debug"`

	Log telegraf.Logger

	deviceInfos  map[string]*deviceInfo
	cachedClient *http.Client
	queryCounter int
}

func NewFritzBox() *FritzBox {
	return &FritzBox{
		Devices:        [][]string{{"fritz.box", "", ""}},
		Timeout:        5,
		GetDeviceInfo:  true,
		GetWLANInfo:    true,
		GetWANInfo:     true,
		GetDSLInfo:     true,
		GetPPPInfo:     true,
		GetMeshInfo:    []string{},
		FullQueryCycle: 6,

		deviceInfos: make(map[string]*deviceInfo)}
}

func (plugin *FritzBox) SampleConfig() string {
	return `
  ## The fritz devices to query (multiple triples of base url, login, password)
  devices = [["http://fritz.box:49000", "", ""]]
  ## The http timeout to use (in seconds)
  # timeout = 5
  ## Skip TLS verification (insecure)
  # tls_skip_verify = false
  ## Process Device services (if found)
  # get_device_info = true
  ## Process WLAN services (if found)
  # get_wlan_info = true
  ## Process WAN services (if found)
  # get_wan_info = true
  ## Process DSL services (if found)
  # get_dsl_info = true
  ## Process PPP services (if found)
  # get_ppp_info = true
  ## Process Mesh infos for selected hosts (must be one of the hosts defined in devices)
  # get_mesh_info = []
  ## Get all mesh clients from mesh infos
  # get_mesh_clients = false
  ## The cycle count, at which low-traffic stats are queried
  # full_query_cycle = 6
  ## Enable debug output
  # debug = false
`
}

func (plugin *FritzBox) Description() string {
	return "Gather FritzBox stats"
}

func (plugin *FritzBox) Gather(a telegraf.Accumulator) error {
	if len(plugin.Devices) == 0 {
		return errors.New("fritzbox: Empty device list")
	}
	for _, device := range plugin.Devices {
		if len(device) != 3 {
			return fmt.Errorf("fritzbox: Invalid device entry: %s", device)
		}
		rawBaseUrl := device[0]
		login := device[1]
		password := device[2]
		deviceInfo, err := plugin.fetchDeviceInfo(rawBaseUrl, login, password)
		if err == nil {
			a.AddError(plugin.processRootDevice(a, deviceInfo))
		} else {
			a.AddError(err)
		}
	}
	plugin.queryCounter++
	if 1 < plugin.FullQueryCycle {
		plugin.queryCounter %= plugin.FullQueryCycle
	} else {
		plugin.queryCounter %= 1
	}
	return nil
}

func (plugin *FritzBox) processRootDevice(a telegraf.Accumulator, deviceInfo *deviceInfo) error {
	if plugin.Debug {
		plugin.Log.Infof("Considering root device: %s", deviceInfo.ServiceInfo.FriendlyName)
	}
	plugin.processServices(a, deviceInfo, deviceInfo.ServiceInfo.Services)
	plugin.processDevices(a, deviceInfo, deviceInfo.ServiceInfo.Devices)
	return nil
}

func (plugin *FritzBox) processDevices(a telegraf.Accumulator, deviceInfo *deviceInfo, devices []tr64DescDevice) error {
	for _, device := range devices {
		if plugin.Debug {
			plugin.Log.Infof("Considering device: %s", device.FriendlyName)
		}
		plugin.processServices(a, deviceInfo, device.Services)
		plugin.processDevices(a, deviceInfo, device.Devices)
	}
	return nil
}

func (plugin *FritzBox) processServices(a telegraf.Accumulator, deviceInfo *deviceInfo, services []tr64DescDeviceService) error {
	for _, service := range services {
		if plugin.Debug {
			plugin.Log.Infof("Considering service type: %s", service.ServiceType)
		}
		fullQuery := plugin.queryCounter == 0
		if strings.HasPrefix(service.ServiceType, "urn:dslforum-org:service:DeviceInfo:") {
			if plugin.GetDeviceInfo && fullQuery {
				a.AddError(plugin.processDeviceInfoService(a, deviceInfo, &service))
			}
		} else if strings.HasPrefix(service.ServiceType, "urn:dslforum-org:service:WLANConfiguration:") {
			if plugin.GetWLANInfo && fullQuery {
				a.AddError(plugin.processWLANConfigurationService(a, deviceInfo, &service))
			}
		} else if strings.HasPrefix(service.ServiceType, "urn:dslforum-org:service:WANCommonInterfaceConfig:") {
			if plugin.GetWANInfo {
				a.AddError(plugin.processWANCommonInterfaceConfigService(a, deviceInfo, &service))
			}
		} else if strings.HasPrefix(service.ServiceType, "urn:dslforum-org:service:WANDSLInterfaceConfig:") {
			if plugin.GetDSLInfo && fullQuery {
				a.AddError(plugin.processDSLInterfaceConfigService(a, deviceInfo, &service))
			}
		} else if strings.HasPrefix(service.ServiceType, "urn:dslforum-org:service:WANPPPConnection:") {
			if plugin.GetPPPInfo && fullQuery {
				a.AddError(plugin.processPPPConnectionService(a, deviceInfo, &service))
			}
		} else if strings.HasPrefix(service.ServiceType, "urn:dslforum-org:service:Hosts:") {
			if deviceInfo.GetMeshInfo && fullQuery {
				a.AddError(plugin.processHostsMeshService(a, deviceInfo, &service))
			}
		}

	}
	return nil
}

func (plugin *FritzBox) processDeviceInfoService(a telegraf.Accumulator, deviceInfo *deviceInfo, service *tr64DescDeviceService) error {
	info := struct {
		UpTime    uint   `xml:"Body>GetInfoResponse>NewUpTime"`
		ModelName string `xml:"Body>GetInfoResponse>NewModelName"`
	}{}
	err := plugin.invokeDeviceService(deviceInfo, service, "GetInfo", &info)
	if err != nil {
		return err
	}
	tags := make(map[string]string)
	tags["fritz_device"] = deviceInfo.BaseUrl.Hostname()
	tags["fritz_service"] = service.ShortServiceId()
	fields := make(map[string]interface{})
	fields["uptime"] = info.UpTime
	fields["model_name"] = info.ModelName
	a.AddCounter("fritzbox_device", fields, tags)
	return nil
}

func (plugin *FritzBox) processWLANConfigurationService(a telegraf.Accumulator, deviceInfo *deviceInfo, service *tr64DescDeviceService) error {
	info := struct {
		Status  string `xml:"Body>GetInfoResponse>NewStatus"`
		Channel string `xml:"Body>GetInfoResponse>NewChannel"`
		SSID    string `xml:"Body>GetInfoResponse>NewSSID"`
	}{}
	err := plugin.invokeDeviceService(deviceInfo, service, "GetInfo", &info)
	if err != nil {
		return err
	}
	totalAssociations := struct {
		TotalAssociations uint `xml:"Body>GetTotalAssociationsResponse>NewTotalAssociations"`
	}{}
	err = plugin.invokeDeviceService(deviceInfo, service, "GetTotalAssociations", &totalAssociations)
	if err != nil {
		return err
	}
	if info.Status == "Up" {
		tags := make(map[string]string)
		tags["fritz_device"] = deviceInfo.BaseUrl.Hostname()
		tags["fritz_service"] = service.ShortServiceId()
		tags["fritz_wlan_channel"] = deviceInfo.BaseUrl.Hostname() + ":" + info.SSID + ":" + info.Channel
		tags["fritz_wlan_network"] = deviceInfo.BaseUrl.Hostname() + ":" + info.SSID + ":" + getNetworkFromChannel(info.Channel)
		fields := make(map[string]interface{})
		fields["total_associations"] = totalAssociations.TotalAssociations
		a.AddCounter("fritzbox_wlan", fields, tags)
	}
	return nil
}

func getNetworkFromChannel(channel string) string {
	if strings.Contains("1 2 3 4 5 6 7 8 9 10 11 12 13 14", channel) {
		return "2G"
	}
	return "5G"
}

func (plugin *FritzBox) processWANCommonInterfaceConfigService(a telegraf.Accumulator, deviceInfo *deviceInfo, service *tr64DescDeviceService) error {
	commonLinkProperties := struct {
		Layer1UpstreamMaxBitRate   uint   `xml:"Body>GetCommonLinkPropertiesResponse>NewLayer1UpstreamMaxBitRate"`
		Layer1DownstreamMaxBitRate uint   `xml:"Body>GetCommonLinkPropertiesResponse>NewLayer1DownstreamMaxBitRate"`
		PhysicalLinkStatus         string `xml:"Body>GetCommonLinkPropertiesResponse>NewPhysicalLinkStatus"`
		UpstreamCurrentMaxSpeed    uint   `xml:"Body>GetCommonLinkPropertiesResponse>NewX_AVM-DE_UpstreamCurrentMaxSpeed"`
		DownstreamCurrentMaxSpeed  uint   `xml:"Body>GetCommonLinkPropertiesResponse>NewX_AVM-DE_DownstreamCurrentMaxSpeed"`
	}{}
	err := plugin.invokeDeviceService(deviceInfo, service, "GetCommonLinkProperties", &commonLinkProperties)
	if err != nil {
		return err
	}
	// Use public IGD service instead of the found one, because IGD supports uint8 counters
	igdWANCommonInterfaceConfigService := tr64DescDeviceService{
		ServiceType: "urn:schemas-upnp-org:service:WANCommonInterfaceConfig:1",
		ServiceId:   "urn:schemas-upnp-org:service:WANCommonInterfaceConfig:1",
		ControlURL:  "/igdupnp/control/WANCommonIFC1"}
	addonInfos := struct {
		ByteSendRate         uint   `xml:"Body>GetAddonInfosResponse>NewByteSendRate"`
		ByteReceiveRate      uint   `xml:"Body>GetAddonInfosResponse>NewByteReceiveRate"`
		TotalBytesSent64     uint64 `xml:"Body>GetAddonInfosResponse>NewX_AVM_DE_TotalBytesSent64"`
		TotalBytesReceived64 uint64 `xml:"Body>GetAddonInfosResponse>NewX_AVM_DE_TotalBytesReceived64"`
	}{}
	err = plugin.invokeDeviceService(deviceInfo, &igdWANCommonInterfaceConfigService, "GetAddonInfos", &addonInfos)
	if err != nil {
		return err
	}
	//totalBytesSent := struct {
	//	TotalBytesSent uint `xml:"Body>GetTotalBytesSentResponse>NewTotalBytesSent"`
	//}{}
	//err = plugin.invokeDeviceService(deviceInfo, service, "GetTotalBytesSent", &totalBytesSent)
	//if err != nil {
	//	return err
	//}
	//totalBytesReceived := struct {
	//	TotalBytesReceived uint `xml:"Body>GetTotalBytesReceivedResponse>NewTotalBytesReceived"`
	//}{}
	//err = plugin.invokeDeviceService(deviceInfo, service, "GetTotalBytesReceived", &totalBytesReceived)
	//if err != nil {
	//	return err
	//}
	if commonLinkProperties.PhysicalLinkStatus == "Up" {
		tags := make(map[string]string)
		tags["fritz_device"] = deviceInfo.BaseUrl.Hostname()
		tags["fritz_service"] = service.ShortServiceId()
		fields := make(map[string]interface{})
		fields["layer1_upstream_max_bit_rate"] = commonLinkProperties.Layer1UpstreamMaxBitRate
		fields["layer1_downstream_max_bit_rate"] = commonLinkProperties.Layer1DownstreamMaxBitRate
		fields["upstream_current_max_speed"] = commonLinkProperties.UpstreamCurrentMaxSpeed
		fields["downstream_current_max_speed"] = commonLinkProperties.DownstreamCurrentMaxSpeed
		//	fields["byte_send_rate"] = addonInfos.ByteSendRate
		//	fields["byte_receive_rate"] = addonInfos.ByteReceiveRate
		fields["total_bytes_sent"] = addonInfos.TotalBytesSent64
		fields["total_bytes_received"] = addonInfos.TotalBytesReceived64
		a.AddCounter("fritzbox_wan", fields, tags)
	}
	return nil
}

func (plugin *FritzBox) processDSLInterfaceConfigService(a telegraf.Accumulator, deviceInfo *deviceInfo, service *tr64DescDeviceService) error {
	info := struct {
		Status                string `xml:"Body>GetInfoResponse>NewStatus"`
		UpstreamCurrRate      uint   `xml:"Body>GetInfoResponse>NewUpstreamCurrRate"`
		DownstreamCurrRate    uint   `xml:"Body>GetInfoResponse>NewDownstreamCurrRate"`
		UpstreamMaxRate       uint   `xml:"Body>GetInfoResponse>NewUpstreamMaxRate"`
		DownstreamMaxRate     uint   `xml:"Body>GetInfoResponse>NewDownstreamMaxRate"`
		UpstreamNoiseMargin   uint   `xml:"Body>GetInfoResponse>NewUpstreamNoiseMargin"`
		DownstreamNoiseMargin uint   `xml:"Body>GetInfoResponse>NewDownstreamNoiseMargin"`
		UpstreamAttenuation   uint   `xml:"Body>GetInfoResponse>NewUpstreamAttenuation"`
		DownstreamAttenuation uint   `xml:"Body>GetInfoResponse>NewDownstreamAttenuation"`
		UpstreamPower         uint   `xml:"Body>GetInfoResponse>NewUpstreamPower"`
		DownstreamPower       uint   `xml:"Body>GetInfoResponse>NewDownstreamPower"`
	}{}
	err := plugin.invokeDeviceService(deviceInfo, service, "GetInfo", &info)
	if err != nil {
		return err
	}
	statisticsTotal := struct {
		ReceiveBlocks       uint `xml:"Body>GetStatisticsTotalResponse>NewReceiveBlocks"`
		TransmitBlocks      uint `xml:"Body>GetStatisticsTotalResponse>NewTransmitBlocks"`
		CellDelin           uint `xml:"Body>GetStatisticsTotalResponse>NewCellDelin"`
		LinkRetrain         uint `xml:"Body>GetStatisticsTotalResponse>NewLinkRetrain"`
		InitErrors          uint `xml:"Body>GetStatisticsTotalResponse>NewInitErrors"`
		InitTimeouts        uint `xml:"Body>GetStatisticsTotalResponse>NewInitTimeouts"`
		LossOfFraming       uint `xml:"Body>GetStatisticsTotalResponse>NewLossOfFraming"`
		ErroredSecs         uint `xml:"Body>GetStatisticsTotalResponse>NewErroredSecs"`
		SeverelyErroredSecs uint `xml:"Body>GetStatisticsTotalResponse>NewSeverelyErroredSecs"`
		FECErrors           uint `xml:"Body>GetStatisticsTotalResponse>NewFECErrors"`
		ATUCFECErrors       uint `xml:"Body>GetStatisticsTotalResponse>NewATUCFECErrors"`
		HECErrors           uint `xml:"Body>GetStatisticsTotalResponse>NewHECErrors"`
		ATUCHECErrors       uint `xml:"Body>GetStatisticsTotalResponse>NewATUCHECErrors"`
		CRCErrors           uint `xml:"Body>GetStatisticsTotalResponse>NewCRCErrors"`
		ATUCCRCErrors       uint `xml:"Body>GetStatisticsTotalResponse>NewATUCCRCErrors"`
	}{}
	err = plugin.invokeDeviceService(deviceInfo, service, "GetStatisticsTotal", &statisticsTotal)
	if err != nil {
		return err
	}
	if info.Status == "Up" {
		tags := make(map[string]string)
		tags["fritz_device"] = deviceInfo.BaseUrl.Hostname()
		tags["fritz_service"] = service.ShortServiceId()
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

func (plugin *FritzBox) processPPPConnectionService(a telegraf.Accumulator, deviceInfo *deviceInfo, service *tr64DescDeviceService) error {
	info := struct {
		ConnectionStatus     string `xml:"Body>GetInfoResponse>NewConnectionStatus"`
		Uptime               uint   `xml:"Body>GetInfoResponse>NewUptime"`
		UpstreamMaxBitRate   uint   `xml:"Body>GetInfoResponse>NewUpstreamMaxBitRate"`
		DownstreamMaxBitRate uint   `xml:"Body>GetInfoResponse>NewDownstreamMaxBitRate"`
	}{}
	err := plugin.invokeDeviceService(deviceInfo, service, "GetInfo", &info)
	if err != nil {
		return err
	}
	if info.ConnectionStatus == "Connected" {
		tags := make(map[string]string)
		tags["fritz_device"] = deviceInfo.BaseUrl.Hostname()
		tags["fritz_service"] = service.ShortServiceId()
		fields := make(map[string]interface{})
		fields["uptime"] = info.Uptime
		fields["upstream_max_bit_rate"] = info.UpstreamMaxBitRate
		fields["downstream_max_bit_rate"] = info.DownstreamMaxBitRate
		a.AddCounter("fritzbox_ppp", fields, tags)
	}
	return nil
}

func (plugin *FritzBox) processHostsMeshService(a telegraf.Accumulator, deviceInfo *deviceInfo, service *tr64DescDeviceService) error {
	meshListPath := struct {
		MeshListPath string `xml:"Body>X_AVM-DE_GetMeshListPathResponse>NewX_AVM-DE_MeshListPath"`
	}{}
	err := plugin.invokeDeviceService(deviceInfo, service, "X_AVM-DE_GetMeshListPath", &meshListPath)
	if err != nil {
		return err
	}

	var meshList meshList

	_, err = plugin.fetchJSON(deviceInfo.BaseUrl, meshListPath.MeshListPath, &meshList)
	if err != nil {
		return err
	}

	masterSlavePaths := meshList.getMasterSlavePaths()
	for _, masterSlavePath := range masterSlavePaths {
		tags := make(map[string]string)
		tags["fritz_device"] = deviceInfo.BaseUrl.Hostname()
		tags["fritz_service"] = service.ShortServiceId()
		tags["fritz_mesh_node_name"] = masterSlavePath.node.DeviceName
		tags["fritz_mesh_node_type"] = masterSlavePath.nodeInterface.Type
		tags["fritz_mesh_node_link"] = masterSlavePath.node.DeviceName + ":" + masterSlavePath.nodeInterface.Type + ":" + masterSlavePath.nodeInterface.Name
		fields := make(map[string]interface{})
		masterSlaveDataRates := masterSlavePath.getRoot().getDataRates()
		fields["max_data_rate_rx"] = masterSlaveDataRates[0]
		fields["max_data_rate_tx"] = masterSlaveDataRates[1]
		fields["cur_data_rate_rx"] = masterSlaveDataRates[2]
		fields["cur_data_rate_tx"] = masterSlaveDataRates[3]
		a.AddCounter("fritzbox_mesh", fields, tags)
	}
	if plugin.GetMeshClients {
		clientPaths := meshList.getClientPaths()
		for _, clientPath := range clientPaths {
			tags := make(map[string]string)
			peer := clientPath.getRoot()
			tags["fritz_device"] = deviceInfo.BaseUrl.Hostname()
			tags["fritz_service"] = service.ShortServiceId()
			tags["fritz_mesh_client_name"] = clientPath.node.DeviceName
			tags["fritz_mesh_client_peer"] = peer.node.DeviceName
			tags["fritz_mesh_client_link"] = peer.nodeInterface.Name
			fields := make(map[string]interface{})
			clientDataRates := clientPath.getDataRates()
			fields["max_data_rate_rx"] = clientDataRates[0]
			fields["max_data_rate_tx"] = clientDataRates[1]
			fields["cur_data_rate_rx"] = clientDataRates[2]
			fields["cur_data_rate_tx"] = clientDataRates[3]
			a.AddCounter("fritzbox_mesh_client", fields, tags)
		}
	}
	return nil
}

func (plugin *FritzBox) invokeDeviceService(deviceInfo *deviceInfo, service *tr64DescDeviceService, action string, out interface{}) error {
	controlUrl, err := url.Parse(service.ControlURL)
	if err != nil {
		return err
	}
	endpoint := deviceInfo.BaseUrl.ResolveReference(controlUrl).String()
	soapAction := fmt.Sprintf("%s#%s", service.ServiceType, action)
	requestBody := fmt.Sprintf(
		`<?xml version="1.0" encoding="utf-8" ?>
		<s:Envelope s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/" xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
			<s:Body>
				<u:%s xmlns:u="%s" />
			</s:Body>
		</s:Envelope>`, action, service.ServiceId)
	cachedAuthentication := plugin.getCachedDigestAuthentication(deviceInfo, service.ServiceType)
	response, err := plugin.postSoapActionRequest(endpoint, soapAction, requestBody, cachedAuthentication)
	if err != nil {
		return err
	}
	if response.StatusCode == http.StatusUnauthorized {
		authentication, err := plugin.getDigestAuthentication(response, deviceInfo, service.ServiceType)
		if err == nil {
			response, err = plugin.postSoapActionRequest(endpoint, soapAction, requestBody, authentication)
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
	if plugin.Debug {
		plugin.Log.Infof("Response:\n%s", responseBody)
	}
	err = xml.Unmarshal(responseBody, out)
	if err != nil {
		return err
	}
	return nil
}

func (plugin *FritzBox) postSoapActionRequest(endpoint string, action string, requestBody string, authentication string) (*http.Response, error) {
	if plugin.Debug {
		plugin.Log.Infof("Invoking SOAP action %s on endpoint %s ...", action, endpoint)
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
	client := plugin.getClient()
	response, err := client.Do(request)
	if err != nil {
		return response, err
	}
	if plugin.Debug {
		plugin.Log.Infof("Status code: %d", response.StatusCode)
	}
	return response, nil
}

func (plugin *FritzBox) getCachedDigestAuthentication(deviceInfo *deviceInfo, uri string) string {
	if deviceInfo.cachedAuthentication[0] == uri {
		return deviceInfo.cachedAuthentication[1]
	}
	return ""
}

func (plugin *FritzBox) getDigestAuthentication(challenge *http.Response, deviceInfo *deviceInfo, uri string) (string, error) {
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
	ha1 := plugin.md5Hash(fmt.Sprintf("%s:%s:%s", deviceInfo.Login, digestRealm, deviceInfo.Password))
	ha2 := plugin.md5Hash(fmt.Sprintf("%s:%s", http.MethodPost, uri))
	nonce := challengeValues["nonce"]
	qop := challengeValues["qop"]
	cnonce := plugin.generateCNonce()
	nc := "1"
	response := plugin.md5Hash(fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, nonce, nc, cnonce, qop, ha2))
	authentication := fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", cnonce="%s", nc="%v", qop="%s", response="%s"`,
		deviceInfo.Login, digestRealm, nonce, uri, cnonce, nc, qop, response)
	deviceInfo.cachedAuthentication[0] = uri
	deviceInfo.cachedAuthentication[1] = authentication
	return authentication, nil
}

func (plugin *FritzBox) md5Hash(in string) string {
	hash := md5.New()
	_, err := hash.Write([]byte(in))
	if err != nil {
		plugin.Log.Error(err)
		panic(err)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func (plugin *FritzBox) generateCNonce() string {
	cnonceBytes := make([]byte, 8)
	_, err := io.ReadFull(rand.Reader, cnonceBytes)
	if err != nil {
		plugin.Log.Error(err)
		panic(err)
	}
	return fmt.Sprintf("%016x", cnonceBytes)
}

func (plugin *FritzBox) fetchDeviceInfo(rawBaseUrl string, login string, password string) (*deviceInfo, error) {
	cachedDeviceInfo, cached := plugin.deviceInfos[rawBaseUrl]
	if !cached {
		if plugin.Debug {
			plugin.Log.Infof("Querying device info for: %s", rawBaseUrl)
		}
		baseUrl, err := url.Parse(rawBaseUrl)
		if err != nil {
			return nil, err
		}

		var serviceInfo tr64Desc

		_, err = plugin.fetchXML(baseUrl, "/tr64desc.xml", &serviceInfo)
		if err != nil {
			return nil, err
		}

		var getMeshInfo bool

		for _, meshMaster := range plugin.GetMeshInfo {
			if meshMaster == baseUrl.Hostname() {
				getMeshInfo = true
				break
			}
		}
		cachedDeviceInfo = &deviceInfo{
			BaseUrl:     baseUrl,
			Login:       login,
			Password:    password,
			GetMeshInfo: getMeshInfo,
			ServiceInfo: &serviceInfo}
		plugin.deviceInfos[rawBaseUrl] = cachedDeviceInfo
	}
	return cachedDeviceInfo, nil
}

func (plugin *FritzBox) fetchXML(baseUrl *url.URL, path string, v interface{}) (*url.URL, error) {
	pathUrl, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	xmlUrl := baseUrl.ResolveReference(pathUrl)
	if plugin.Debug {
		plugin.Log.Infof("Fetching XML from: %s", xmlUrl)
	}
	client := plugin.getClient()
	response, err := client.Get(xmlUrl.String())
	if err != nil {
		return xmlUrl, err
	}
	defer response.Body.Close()
	return xmlUrl, xml.NewDecoder(response.Body).Decode(v)
}

func (plugin *FritzBox) fetchJSON(baseUrl *url.URL, path string, v interface{}) (*url.URL, error) {
	pathUrl, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	jsonUrl := baseUrl.ResolveReference(pathUrl)
	if plugin.Debug {
		plugin.Log.Infof("Fetching JSON from: %s", jsonUrl)
	}
	client := plugin.getClient()
	response, err := client.Get(jsonUrl.String())
	if err != nil {
		return jsonUrl, err
	}
	defer response.Body.Close()
	return jsonUrl, json.NewDecoder(response.Body).Decode(v)
}

func (plugin *FritzBox) getClient() *http.Client {
	if plugin.cachedClient == nil {
		transport := &http.Transport{
			ResponseHeaderTimeout: time.Duration(plugin.Timeout) * time.Second,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: plugin.TLSSkipVerify},
		}
		plugin.cachedClient = &http.Client{
			Transport: transport,
			Timeout:   time.Duration(plugin.Timeout) * time.Second,
		}
	}
	return plugin.cachedClient
}

func init() {
	inputs.Add("fritzbox", func() telegraf.Input {
		return NewFritzBox()
	})
}
