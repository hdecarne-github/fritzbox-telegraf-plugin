// fritzbox_test.go
//
// Copyright (C) 2022 Holger de Carne
//
// This software may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
//
package fritzbox

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
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
	testServerHandler := &testServerHandler{Debug: true}
	testServer := httptest.NewServer(testServerHandler)
	defer testServer.Close()
	testServerURL, err := url.Parse(testServer.URL)
	require.NoError(t, err)
	fb := NewFritzBox()
	fb.Devices = [][]string{{testServer.URL, "user", "secret"}}
	fb.GetMeshInfo = []string{testServerURL.Hostname()}
	fb.Log = createDummyLogger()
	fb.Debug = testServerHandler.Debug

	var a testutil.Accumulator

	require.NoError(t, a.GatherError(fb.Gather))
	require.True(t, a.HasMeasurement("fritzbox_device"))
	require.True(t, a.HasMeasurement("fritzbox_wlan"))
	require.True(t, a.HasMeasurement("fritzbox_wan"))
	require.True(t, a.HasMeasurement("fritzbox_dsl"))
	require.True(t, a.HasMeasurement("fritzbox_ppp"))
	require.True(t, a.HasMeasurement("fritzbox_mesh"))
}

func createDummyLogger() *dummyLogger {
	log.SetOutput(os.Stderr)
	return &dummyLogger{}
}

type dummyLogger struct{}

func (l *dummyLogger) Errorf(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func (l *dummyLogger) Error(args ...interface{}) {
	log.Print(args...)
}

func (l *dummyLogger) Debugf(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func (l *dummyLogger) Debug(args ...interface{}) {
	log.Print(args...)
}

func (l *dummyLogger) Warnf(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func (l *dummyLogger) Warn(args ...interface{}) {
	log.Print(args...)
}

func (l *dummyLogger) Infof(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func (l *dummyLogger) Info(args ...interface{}) {
	log.Print(args...)
}

type testServerHandler struct {
	Debug bool
}

func (tsh *testServerHandler) ServeHTTP(out http.ResponseWriter, request *http.Request) {
	requestURL := request.URL.String()
	if tsh.Debug {
		log.Printf("test: request URL: %s", requestURL)
	}
	if request.Method == http.MethodPost && request.Header.Get("Authorization") == "" {
		out.Header().Add("Www-Authenticate", `Digest realm="HTTPS Access",nonce="30492F0B4025DFF7",algorithm=MD5,qop="auth"`)
		out.WriteHeader(http.StatusUnauthorized)
	}
	if requestURL == "/tr64desc.xml" {
		tsh.serveTr64descXML(out)
	} else if requestURL == "/upnp/control/deviceinfo" {
		tsh.serveDeviceInfo(out, request)
	} else if requestURL == "/upnp/control/wlanconfig1" {
		tsh.serveWLANConfig1(out, request)
	} else if requestURL == "/upnp/control/wlanconfig2" {
		tsh.serveWLANConfig2(out, request)
	} else if requestURL == "/upnp/control/wlanconfig3" {
		tsh.serveWLANConfig3(out, request)
	} else if requestURL == "/upnp/control/wancommonifconfig1" {
		tsh.serveWANCommonIfConfig1(out, request)
	} else if requestURL == "/igdupnp/control/WANCommonIFC1" {
		tsh.serveWANCommonIFC1(out, request)
	} else if requestURL == "/upnp/control/wandslifconfig1" {
		tsh.serveWANDSLIfConfig1(out, request)
	} else if requestURL == "/upnp/control/wanpppconn1" {
		tsh.serveWANPPPConn1(out, request)
	} else if requestURL == "/upnp/control/hosts" {
		tsh.serveHosts(out, request)
	} else if requestURL == "/meshlist.lua?sid=9f46d0308fd4fdd9" {
		tsh.serveHostsMeshList(out, request)
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

const testWLANConfig1GetInfoResponse = `
<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
<s:Body>
<u:GetInfoResponse xmlns:u="urn:dslforum-org:service:WLANConfiguration:3">
<NewStatus>Disabled</NewStatus>
<NewChannel>1</NewChannel>
<NewSSID>TestSSID1</NewSSID>
</u:GetInfoResponse>
</s:Body>
</s:Envelope>
`

const testWLANConfig1GetAssociationsResponse = `
<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
<s:Body>
<u:GetTotalAssociationsResponse xmlns:u="urn:dslforum-org:service:WLANConfiguration:3">
<NewTotalAssociations>0</NewTotalAssociations>
</u:GetTotalAssociationsResponse>
</s:Body>
</s:Envelope>
`

func (tsh *testServerHandler) serveWLANConfig1(out http.ResponseWriter, request *http.Request) {
	action := tsh.getSoapAction(request, "urn:WLANConfiguration-com:serviceId:WLANConfiguration1")
	if action == "GetInfo" {
		tsh.writeXML(out, testWLANConfig1GetInfoResponse)
	} else if action == "GetTotalAssociations" {
		tsh.writeXML(out, testWLANConfig1GetAssociationsResponse)
	}
}

const testWLANConfig2GetInfoResponse = `
<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
<s:Body>
<u:GetInfoResponse xmlns:u="urn:dslforum-org:service:WLANConfiguration:3">
<NewStatus>Up</NewStatus>
<NewChannel>2</NewChannel>
<NewSSID>TestSSID2</NewSSID>
</u:GetInfoResponse>
</s:Body>
</s:Envelope>
`

const testWLANConfig2GetAssociationsResponse = `
<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
<s:Body>
<u:GetTotalAssociationsResponse xmlns:u="urn:dslforum-org:service:WLANConfiguration:3">
<NewTotalAssociations>20</NewTotalAssociations>
</u:GetTotalAssociationsResponse>
</s:Body>
</s:Envelope>
`

func (tsh *testServerHandler) serveWLANConfig2(out http.ResponseWriter, request *http.Request) {
	action := tsh.getSoapAction(request, "urn:WLANConfiguration-com:serviceId:WLANConfiguration2")
	if action == "GetInfo" {
		tsh.writeXML(out, testWLANConfig2GetInfoResponse)
	} else if action == "GetTotalAssociations" {
		tsh.writeXML(out, testWLANConfig2GetAssociationsResponse)
	}
}

const testWLANConfig3GetInfoResponse = `
<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
<s:Body>
<u:GetInfoResponse xmlns:u="urn:dslforum-org:service:WLANConfiguration:3">
<NewStatus>Up</NewStatus>
<NewChannel>3</NewChannel>
<NewSSID>TestSSID3</NewSSID>
</u:GetInfoResponse>
</s:Body>
</s:Envelope>
`

const testWLANConfig3GetAssociationsResponse = `
<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
<s:Body>
<u:GetTotalAssociationsResponse xmlns:u="urn:dslforum-org:service:WLANConfiguration:3">
<NewTotalAssociations>30</NewTotalAssociations>
</u:GetTotalAssociationsResponse>
</s:Body>
</s:Envelope>
`

func (tsh *testServerHandler) serveWLANConfig3(out http.ResponseWriter, request *http.Request) {
	action := tsh.getSoapAction(request, "urn:WLANConfiguration-com:serviceId:WLANConfiguration3")
	if action == "GetInfo" {
		tsh.writeXML(out, testWLANConfig3GetInfoResponse)
	} else if action == "GetTotalAssociations" {
		tsh.writeXML(out, testWLANConfig3GetAssociationsResponse)
	}
}

const testWANCommonIfConfig1GetCommonLinkPropertiesResponse = `
<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
<s:Body>
<u:GetCommonLinkPropertiesResponse xmlns:u="urn:dslforum-org:service:WANCommonInterfaceConfig:1">
<NewLayer1UpstreamMaxBitRate>49741000</NewLayer1UpstreamMaxBitRate>
<NewLayer1DownstreamMaxBitRate>240893000</NewLayer1DownstreamMaxBitRate>
<NewPhysicalLinkStatus>Up</NewPhysicalLinkStatus>
<NewX_AVM-DE_DownstreamCurrentMaxSpeed>1711517</NewX_AVM-DE_DownstreamCurrentMaxSpeed>
<NewX_AVM-DE_UpstreamCurrentMaxSpeed>53711</NewX_AVM-DE_UpstreamCurrentMaxSpeed>
</u:GetCommonLinkPropertiesResponse>
</s:Body>
</s:Envelope>
`

func (tsh *testServerHandler) serveWANCommonIfConfig1(out http.ResponseWriter, request *http.Request) {
	action := tsh.getSoapAction(request, "urn:WANCIfConfig-com:serviceId:WANCommonInterfaceConfig1")
	if action == "GetCommonLinkProperties" {
		tsh.writeXML(out, testWANCommonIfConfig1GetCommonLinkPropertiesResponse)
	}
}

const testWANCommonIFC1GetAddonInfosResponse = `
<?xml version="1.0" encoding="utf-8"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
<s:Body>
<u:GetAddonInfosResponse xmlns:u="urn:schemas-upnp-org:service:WANCommonInterfaceConfig:1">
<NewByteSendRate>79128</NewByteSendRate>
<NewByteReceiveRate>1148054</NewByteReceiveRate>
<NewTotalBytesSent>615295140</NewTotalBytesSent>
<NewTotalBytesReceived>217715745</NewTotalBytesReceived>
<NewX_AVM_DE_TotalBytesSent64>30680066212</NewX_AVM_DE_TotalBytesSent64>
<NewX_AVM_DE_TotalBytesReceived64>197786211361</NewX_AVM_DE_TotalBytesReceived64>
</u:GetAddonInfosResponse>
</s:Body>
</s:Envelope>
`

func (tsh *testServerHandler) serveWANCommonIFC1(out http.ResponseWriter, request *http.Request) {
	action := tsh.getSoapAction(request, "urn:schemas-upnp-org:service:WANCommonInterfaceConfig:1")
	if action == "GetAddonInfos" {
		tsh.writeXML(out, testWANCommonIFC1GetAddonInfosResponse)
	}
}

const testWANDSLIfConfig1GetInfoResponse = `
<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
<s:Body>
<u:GetInfoResponse xmlns:u="urn:dslforum-org:service:WANDSLInterfaceConfig:1">
<NewStatus>Up</NewStatus>
<NewUpstreamCurrRate>46719</NewUpstreamCurrRate>
<NewDownstreamCurrRate>236716</NewDownstreamCurrRate>
<NewUpstreamMaxRate>49741</NewUpstreamMaxRate>
<NewDownstreamMaxRate>240893</NewDownstreamMaxRate>
<NewUpstreamNoiseMargin>80</NewUpstreamNoiseMargin>
<NewDownstreamNoiseMargin>110</NewDownstreamNoiseMargin>
<NewUpstreamAttenuation>80</NewUpstreamAttenuation>
<NewDownstreamAttenuation>140</NewDownstreamAttenuation>
<NewUpstreamPower>498</NewUpstreamPower>
<NewDownstreamPower>515</NewDownstreamPower>
</u:GetInfoResponse>
</s:Body>
</s:Envelope>
`

const testWANDSLIfConfig1GetStatisticsTotalResponse = `
<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
<s:Body>
<u:GetStatisticsTotalResponse xmlns:u="urn:dslforum-org:service:WANDSLInterfaceConfig:1">
<NewCellDelin>0</NewCellDelin>
<NewLinkRetrain>1</NewLinkRetrain>
<NewInitErrors>0</NewInitErrors>
<NewInitTimeouts>0</NewInitTimeouts>
<NewLossOfFraming>0</NewLossOfFraming>
<NewErroredSecs>4</NewErroredSecs>
<NewSeverelyErroredSecs>0</NewSeverelyErroredSecs>
<NewFECErrors>0</NewFECErrors>
<NewATUCFECErrors>0</NewATUCFECErrors>
<NewHECErrors>0</NewHECErrors>
<NewATUCHECErrors>0</NewATUCHECErrors>
<NewCRCErrors>6</NewCRCErrors>
<NewATUCCRCErrors>1</NewATUCCRCErrors>
</u:GetStatisticsTotalResponse>
</s:Body>
</s:Envelope>
`

func (tsh *testServerHandler) serveWANDSLIfConfig1(out http.ResponseWriter, request *http.Request) {
	action := tsh.getSoapAction(request, "urn:WANDSLIfConfig-com:serviceId:WANDSLInterfaceConfig1")
	if action == "GetInfo" {
		tsh.writeXML(out, testWANDSLIfConfig1GetInfoResponse)
	} else if action == "GetStatisticsTotal" {
		tsh.writeXML(out, testWANDSLIfConfig1GetStatisticsTotalResponse)
	}
}

const testWANPPPConn1GetInfoResponse = `
<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
<s:Body>
<u:GetInfoResponse xmlns:u="urn:dslforum-org:service:WANPPPConnection:1">
<NewConnectionStatus>Connected</NewConnectionStatus>
<NewUptime>755581</NewUptime>
<NewUpstreamMaxBitRate>45048452</NewUpstreamMaxBitRate>
<NewDownstreamMaxBitRate>56093007</NewDownstreamMaxBitRate>
</u:GetInfoResponse>
</s:Body>
</s:Envelope>
`

func (tsh *testServerHandler) serveWANPPPConn1(out http.ResponseWriter, request *http.Request) {
	action := tsh.getSoapAction(request, "urn:WANPPPConnection-com:serviceId:WANPPPConnection1")
	if action == "GetInfo" {
		tsh.writeXML(out, testWANPPPConn1GetInfoResponse)
	}
}

const testHostsGetMeshListPath = `
<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
<s:Body>
<u:X_AVM-DE_GetMeshListPathResponse xmlns:u="urn:dslforum-org:service:Hosts:1">
<NewX_AVM-DE_MeshListPath>/meshlist.lua?sid=9f46d0308fd4fdd9</NewX_AVM-DE_MeshListPath>
</u:X_AVM-DE_GetMeshListPathResponse>
</s:Body>
</s:Envelope>
`

func (tsh *testServerHandler) serveHosts(out http.ResponseWriter, request *http.Request) {
	action := tsh.getSoapAction(request, "urn:LanDeviceHosts-com:serviceId:Hosts1")
	if action == "X_AVM-DE_GetMeshListPath" {
		tsh.writeXML(out, testHostsGetMeshListPath)
	}
}

const testHostsMeshList = `
{
	"schema_version": "4.7",
	"nodes": [
		{
			"uid": "n-1",
			"device_name": "master1",
			"is_meshed": true,
			"mesh_role": "master",
			"node_interfaces": [

			]
		},
		{
			"uid": "n-145",
			"device_name": "slave1",
			"is_meshed": true,
			"mesh_role": "slave",
			"node_interfaces": [
				{
					"name": "UPLINK:5G:0",
					"type": "WLAN",
					"node_links": [
						{
							"state": "CONNECTED",
							"node_1_uid": "n-1",
							"node_2_uid": "n-145",
							"max_data_rate_rx": 1300000,
							"max_data_rate_tx": 1300000,
							"cur_data_rate_rx": 1300000,
							"cur_data_rate_tx": 975000
						}
					]
				},
				{
					"name": "UPLINK:2G:0",
					"type": "WLAN",
					"node_links": [
						{
							"state": "CONNECTED",
							"node_1_uid": "n-1",
							"node_2_uid": "n-145",
							"max_data_rate_rx": 216000,
							"max_data_rate_tx": 216000,
							"cur_data_rate_rx": 216000,
							"cur_data_rate_tx": 216000
						}
					]
				}
			]
		}
	]
}
`

func (tsh *testServerHandler) serveHostsMeshList(out http.ResponseWriter, request *http.Request) {
	tsh.writeJSON(out, testHostsMeshList)
}

func (tsh *testServerHandler) getSoapAction(request *http.Request, uri string) string {
	matcher := regexp.MustCompile(fmt.Sprintf(`(?s)<u:(.*) xmlns:u="%s" />`, uri))
	defer request.Body.Close()
	body, _ := io.ReadAll(request.Body)
	if tsh.Debug {
		log.Printf("Request body:\n%s", body)
	}
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

func (tsh *testServerHandler) writeJSON(out http.ResponseWriter, json string) {
	out.Header().Add("Content-Type", "application/json")
	_, _ = out.Write([]byte(json))
}
