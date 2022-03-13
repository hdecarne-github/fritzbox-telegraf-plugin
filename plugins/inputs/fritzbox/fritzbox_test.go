package fritzbox

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/influxdata/telegraf/testutil"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	fb := NewFritzBox()
	require.NotNil(t, fb)
}

func TestSampleConfig(t *testing.T) {
	fb := NewFritzBox()
	sampleConfig := fb.SampleConfig()
	require.NotNil(t, sampleConfig)
}

func TestDescription(t *testing.T) {
	fb := NewFritzBox()
	description := fb.Description()
	require.NotNil(t, description)
}

func TestGather1(t *testing.T) {
	testServer := httptest.NewServer(&testServerHandler{})
	defer testServer.Close()
	fb := NewFritzBox()
	fb.GetWLANInfo = false
	fb.GetWANInfo = false
	fb.GetDSLInfo = false
	fb.GetPPPInfo = false
	fb.Devices = [][]string{{testServer.URL, "", ""}}
	var a testutil.Accumulator
	require.NoError(t, a.GatherError(fb.Gather))
}

type testServerHandler struct {
}

func (tsh *testServerHandler) ServeHTTP(out http.ResponseWriter, request *http.Request) {
	requestURL := request.URL.String()
	log.Printf("test: request URL: %s", requestURL)
	if requestURL == "/tr64desc.xml" {
		tsh.serveTr64descXML(out)
	} else if requestURL == "/upnp/control/deviceinfo" {
		tsh.serveDeviceInfo(out, request)
	} else if requestURL == "/upnp/control/wlanconfig1" {

	} else if requestURL == "/upnp/control/wlanconfig2" {

	} else if requestURL == "/upnp/control/wlanconfig3" {

	} else if requestURL == "/upnp/control/wancommonifconfig1" {

	} else if requestURL == "/upnp/control/wandslifconfig1" {

	} else if requestURL == "/upnp/control/wanpppconn1" {

	}
}

const testTr64descXML = `
<root xmlns="urn:dslforum-org:device-1-0">
<device>
<friendlyName>Test Device 1</friendlyName>
<serviceList>
<service>
<serviceType>urn:dslforum-org:service:DeviceInfo:1</serviceType>
<serviceId>urn:DeviceInfo-com:serviceId:DeviceInfo1</serviceId>
<controlURL>/upnp/control/deviceinfo</controlURL>
<eventSubURL>/upnp/control/deviceinfo</eventSubURL>
<SCPDURL>/deviceinfoSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:DeviceConfig:1</serviceType>
<serviceId>urn:DeviceConfig-com:serviceId:DeviceConfig1</serviceId>
<controlURL>/upnp/control/deviceconfig</controlURL>
<eventSubURL>/upnp/control/deviceconfig</eventSubURL>
<SCPDURL>/deviceconfigSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:Layer3Forwarding:1</serviceType>
<serviceId>urn:Layer3Forwarding-com:serviceId:Layer3Forwarding1</serviceId>
<controlURL>/upnp/control/layer3forwarding</controlURL>
<eventSubURL>/upnp/control/layer3forwarding</eventSubURL>
<SCPDURL>/layer3forwardingSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:LANConfigSecurity:1</serviceType>
<serviceId>urn:LANConfigSecurity-com:serviceId:LANConfigSecurity1</serviceId>
<controlURL>/upnp/control/lanconfigsecurity</controlURL>
<eventSubURL>/upnp/control/lanconfigsecurity</eventSubURL>
<SCPDURL>/lanconfigsecuritySCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:ManagementServer:1</serviceType>
<serviceId>urn:ManagementServer-com:serviceId:ManagementServer1</serviceId>
<controlURL>/upnp/control/mgmsrv</controlURL>
<eventSubURL>/upnp/control/mgmsrv</eventSubURL>
<SCPDURL>/mgmsrvSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:Time:1</serviceType>
<serviceId>urn:Time-com:serviceId:Time1</serviceId>
<controlURL>/upnp/control/time</controlURL>
<eventSubURL>/upnp/control/time</eventSubURL>
<SCPDURL>/timeSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:UserInterface:1</serviceType>
<serviceId>urn:UserInterface-com:serviceId:UserInterface1</serviceId>
<controlURL>/upnp/control/userif</controlURL>
<eventSubURL>/upnp/control/userif</eventSubURL>
<SCPDURL>/userifSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:X_AVM-DE_Storage:1</serviceType>
<serviceId>urn:X_AVM-DE_Storage-com:serviceId:X_AVM-DE_Storage1</serviceId>
<controlURL>/upnp/control/x_storage</controlURL>
<eventSubURL>/upnp/control/x_storage</eventSubURL>
<SCPDURL>/x_storageSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:X_AVM-DE_WebDAVClient:1</serviceType>
<serviceId>urn:X_AVM-DE_WebDAV-com:serviceId:X_AVM-DE_WebDAVClient1</serviceId>
<controlURL>/upnp/control/x_webdav</controlURL>
<eventSubURL>/upnp/control/x_webdav</eventSubURL>
<SCPDURL>/x_webdavSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:X_AVM-DE_UPnP:1</serviceType>
<serviceId>urn:X_AVM-DE_UPnP-com:serviceId:X_AVM-DE_UPnP1</serviceId>
<controlURL>/upnp/control/x_upnp</controlURL>
<eventSubURL>/upnp/control/x_upnp</eventSubURL>
<SCPDURL>/x_upnpSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:X_AVM-DE_Speedtest:1</serviceType>
<serviceId>urn:X_AVM-DE_Speedtest-com:serviceId:X_AVM-DE_Speedtest1</serviceId>
<controlURL>/upnp/control/x_speedtest</controlURL>
<eventSubURL>/upnp/control/x_speedtest</eventSubURL>
<SCPDURL>/x_speedtestSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:X_AVM-DE_RemoteAccess:1</serviceType>
<serviceId>urn:X_AVM-DE_RemoteAccess-com:serviceId:X_AVM-DE_RemoteAccess1</serviceId>
<controlURL>/upnp/control/x_remote</controlURL>
<eventSubURL>/upnp/control/x_remote</eventSubURL>
<SCPDURL>/x_remoteSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:X_AVM-DE_MyFritz:1</serviceType>
<serviceId>urn:X_AVM-DE_MyFritz-com:serviceId:X_AVM-DE_MyFritz1</serviceId>
<controlURL>/upnp/control/x_myfritz</controlURL>
<eventSubURL>/upnp/control/x_myfritz</eventSubURL>
<SCPDURL>/x_myfritzSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:X_VoIP:1</serviceType>
<serviceId>urn:X_VoIP-com:serviceId:X_VoIP1</serviceId>
<controlURL>/upnp/control/x_voip</controlURL>
<eventSubURL>/upnp/control/x_voip</eventSubURL>
<SCPDURL>/x_voipSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:X_AVM-DE_OnTel:1</serviceType>
<serviceId>urn:X_AVM-DE_OnTel-com:serviceId:X_AVM-DE_OnTel1</serviceId>
<controlURL>/upnp/control/x_contact</controlURL>
<eventSubURL>/upnp/control/x_contact</eventSubURL>
<SCPDURL>/x_contactSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:X_AVM-DE_Dect:1</serviceType>
<serviceId>urn:X_AVM-DE_Dect-com:serviceId:X_AVM-DE_Dect1</serviceId>
<controlURL>/upnp/control/x_dect</controlURL>
<eventSubURL>/upnp/control/x_dect</eventSubURL>
<SCPDURL>/x_dectSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:X_AVM-DE_TAM:1</serviceType>
<serviceId>urn:X_AVM-DE_TAM-com:serviceId:X_AVM-DE_TAM1</serviceId>
<controlURL>/upnp/control/x_tam</controlURL>
<eventSubURL>/upnp/control/x_tam</eventSubURL>
<SCPDURL>/x_tamSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:X_AVM-DE_AppSetup:1</serviceType>
<serviceId>urn:X_AVM-DE_AppSetup-com:serviceId:X_AVM-DE_AppSetup1</serviceId>
<controlURL>/upnp/control/x_appsetup</controlURL>
<eventSubURL>/upnp/control/x_appsetup</eventSubURL>
<SCPDURL>/x_appsetupSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:X_AVM-DE_Homeauto:1</serviceType>
<serviceId>urn:X_AVM-DE_Homeauto-com:serviceId:X_AVM-DE_Homeauto1</serviceId>
<controlURL>/upnp/control/x_homeauto</controlURL>
<eventSubURL>/upnp/control/x_homeauto</eventSubURL>
<SCPDURL>/x_homeautoSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:X_AVM-DE_Homeplug:1</serviceType>
<serviceId>urn:X_AVM-DE_Homeplug-com:serviceId:X_AVM-DE_Homeplug1</serviceId>
<controlURL>/upnp/control/x_homeplug</controlURL>
<eventSubURL>/upnp/control/x_homeplug</eventSubURL>
<SCPDURL>/x_homeplugSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:X_AVM-DE_Filelinks:1</serviceType>
<serviceId>urn:X_AVM-DE_Filelinks-com:serviceId:X_AVM-DE_Filelinks1</serviceId>
<controlURL>/upnp/control/x_filelinks</controlURL>
<eventSubURL>/upnp/control/x_filelinks</eventSubURL>
<SCPDURL>/x_filelinksSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:X_AVM-DE_Auth:1</serviceType>
<serviceId>urn:X_AVM-DE_Auth-com:serviceId:X_AVM-DE_Auth1</serviceId>
<controlURL>/upnp/control/x_auth</controlURL>
<eventSubURL>/upnp/control/x_auth</eventSubURL>
<SCPDURL>/x_authSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:X_AVM-DE_HostFilter:1</serviceType>
<serviceId>urn:X_AVM-DE_HostFilter-com:serviceId:X_AVM-DE_HostFilter1</serviceId>
<controlURL>/upnp/control/x_hostfilter</controlURL>
<eventSubURL>/upnp/control/x_hostfilter</eventSubURL>
<SCPDURL>/x_hostfilterSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:X_AVM-DE_USPController:1</serviceType>
<serviceId>urn:X_AVM-DE_USPController-com:serviceId:X_AVM-DE_USPController1</serviceId>
<controlURL>/upnp/control/x_uspcontroller</controlURL>
<eventSubURL>/upnp/control/x_uspcontroller</eventSubURL>
<SCPDURL>/x_uspcontrollerSCPD.xml</SCPDURL>
</service>
</serviceList>
<deviceList>
<device>
<friendlyName>Test Device 2</friendlyName>
<serviceList>
<service>
<serviceType>urn:dslforum-org:service:WLANConfiguration:1</serviceType>
<serviceId>urn:WLANConfiguration-com:serviceId:WLANConfiguration1</serviceId>
<controlURL>/upnp/control/wlanconfig1</controlURL>
<eventSubURL>/upnp/control/wlanconfig1</eventSubURL>
<SCPDURL>/wlanconfigSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:WLANConfiguration:2</serviceType>
<serviceId>urn:WLANConfiguration-com:serviceId:WLANConfiguration2</serviceId>
<controlURL>/upnp/control/wlanconfig2</controlURL>
<eventSubURL>/upnp/control/wlanconfig2</eventSubURL>
<SCPDURL>/wlanconfigSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:WLANConfiguration:3</serviceType>
<serviceId>urn:WLANConfiguration-com:serviceId:WLANConfiguration3</serviceId>
<controlURL>/upnp/control/wlanconfig3</controlURL>
<eventSubURL>/upnp/control/wlanconfig3</eventSubURL>
<SCPDURL>/wlanconfigSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:Hosts:1</serviceType>
<serviceId>urn:LanDeviceHosts-com:serviceId:Hosts1</serviceId>
<controlURL>/upnp/control/hosts</controlURL>
<eventSubURL>/upnp/control/hosts</eventSubURL>
<SCPDURL>/hostsSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:LANEthernetInterfaceConfig:1</serviceType>
<serviceId>urn:LANEthernetIfCfg-com:serviceId:LANEthernetInterfaceConfig1</serviceId>
<controlURL>/upnp/control/lanethernetifcfg</controlURL>
<eventSubURL>/upnp/control/lanethernetifcfg</eventSubURL>
<SCPDURL>/ethifconfigSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:LANHostConfigManagement:1</serviceType>
<serviceId>urn:LANHCfgMgm-com:serviceId:LANHostConfigManagement1</serviceId>
<controlURL>/upnp/control/lanhostconfigmgm</controlURL>
<eventSubURL>/upnp/control/lanhostconfigmgm</eventSubURL>
<SCPDURL>/lanhostconfigmgmSCPD.xml</SCPDURL>
</service>
</serviceList>
</device>
<device>
<friendlyName>Test Device 3</friendlyName>
<serviceList>
<service>
<serviceType>urn:dslforum-org:service:WANCommonInterfaceConfig:1</serviceType>
<serviceId>urn:WANCIfConfig-com:serviceId:WANCommonInterfaceConfig1</serviceId>
<controlURL>/upnp/control/wancommonifconfig1</controlURL>
<eventSubURL>/upnp/control/wancommonifconfig1</eventSubURL>
<SCPDURL>/wancommonifconfigSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:WANDSLInterfaceConfig:1</serviceType>
<serviceId>urn:WANDSLIfConfig-com:serviceId:WANDSLInterfaceConfig1</serviceId>
<controlURL>/upnp/control/wandslifconfig1</controlURL>
<eventSubURL>/upnp/control/wandslifconfig1</eventSubURL>
<SCPDURL>/wandslifconfigSCPD.xml</SCPDURL>
</service>
</serviceList>
<deviceList>
<device>
<friendlyName>Test Device 4</friendlyName>
<serviceList>
<service>
<serviceType>urn:dslforum-org:service:WANDSLLinkConfig:1</serviceType>
<serviceId>urn:WANDSLLinkConfig-com:serviceId:WANDSLLinkConfig1</serviceId>
<controlURL>/upnp/control/wandsllinkconfig1</controlURL>
<eventSubURL>/upnp/control/wandsllinkconfig1</eventSubURL>
<SCPDURL>/wandsllinkconfigSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:WANEthernetLinkConfig:1</serviceType>
<serviceId>urn:WANEthernetLinkConfig-com:serviceId:WANEthernetLinkConfig1</serviceId>
<controlURL>/upnp/control/wanethlinkconfig1</controlURL>
<eventSubURL>/upnp/control/wanethlinkconfig1</eventSubURL>
<SCPDURL>/wanethlinkconfigSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:WANPPPConnection:1</serviceType>
<serviceId>urn:WANPPPConnection-com:serviceId:WANPPPConnection1</serviceId>
<controlURL>/upnp/control/wanpppconn1</controlURL>
<eventSubURL>/upnp/control/wanpppconn1</eventSubURL>
<SCPDURL>/wanpppconnSCPD.xml</SCPDURL>
</service>
<service>
<serviceType>urn:dslforum-org:service:WANIPConnection:1</serviceType>
<serviceId>urn:WANIPConnection-com:serviceId:WANIPConnection1</serviceId>
<controlURL>/upnp/control/wanipconnection1</controlURL>
<eventSubURL>/upnp/control/wanipconnection1</eventSubURL>
<SCPDURL>/wanipconnSCPD.xml</SCPDURL>
</service>
</serviceList>
</device>
</deviceList>
</device>
</deviceList>
</device>
</root>
`

func (tsh *testServerHandler) serveTr64descXML(out http.ResponseWriter) {
	tsh.writeXML(out, testTr64descXML)
}

const testDeviceInfoGetInfoResponse = `
<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
<s:Body>
<u:GetInfoResponse xmlns:u="urn:dslforum-org:service:DeviceInfo:1">
<NewModelName>Test Model 1</NewModelName>
<NewUpTime>751513</NewUpTime>
</u:GetInfoResponse>
</s:Body>
</s:Envelope>
`

func (tsh *testServerHandler) serveDeviceInfo(out http.ResponseWriter, request *http.Request) {
	action := tsh.getSoapAction(request, "urn:DeviceInfo-com:serviceId:DeviceInfo1")
	if action == "GetInfo" {
		tsh.writeXML(out, testDeviceInfoGetInfoResponse)
	}
}

func (tsh *testServerHandler) getSoapAction(request *http.Request, uri string) string {
	matcher := regexp.MustCompile(fmt.Sprintf(`(?s)<u:(.*) xmlns:u="%s" />`, uri))
	defer request.Body.Close()
	body, _ := io.ReadAll(request.Body)
	match := matcher.FindStringSubmatch(string(body))
	if len(match) != 2 {
		return ""
	}
	return match[1]
}

func (tsh *testServerHandler) writeXML(out http.ResponseWriter, xml string) {
	out.Header().Add("Content-Type", "application/xml")
	_, _ = out.Write([]byte(xml))
}
