[![Downloads](https://img.shields.io/github/downloads/hdecarne-github/fritzbox-telegraf-plugin/total.svg)](https://github.com/hdecarne-github/fritzbox-telegraf-plugin/releases)
[![Build](https://github.com/hdecarne-github/fritzbox-telegraf-plugin/actions/workflows/build-on-linux.yml/badge.svg)](https://github.com/hdecarne-github/fritzbox-telegraf-plugin/actions/workflows/build-on-linux.yml)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=hdecarne-github_fritzbox-telegraf-plugin&metric=coverage)](https://sonarcloud.io/summary/new_code?id=hdecarne-github_fritzbox-telegraf-plugin)

## About fritzbox-telegraf-plugin
This [Telegraf](https://github.com/influxdata/telegraf) input plugin gathers stats from [AVM](https://avm.de/) FRITZ!Box devices. It uses the device's [TR-064](https://avm.de/service/schnittstellen/) interfaces to retrieve the stats. DSL routers as well as WLAN repeaters are supported.

### Installation
To install the plugin you have to download the release archive, extract it and build it via a simple
```
make
```
The resulting plugin binary will be written to ./bin. Copy the plugin binary to a location of your choice (e.g. /usr/local/lib/telegraf/)

### Configuration
This is an [external plugin](https://github.com/influxdata/telegraf/blob/master/docs/EXTERNAL_PLUGINS.md) which has to be integrated via Telegraf's [excecd plugin](https://github.com/influxdata/telegraf/tree/master/plugins/inputs/execd.).
To use it you have to create a plugin specific config file (e.g. /etc/telegraf/fritzbox.conf) with following content:
```toml
[[inputs.fritzbox]]
  ## The fritz devices to query (multiple triples of base url, login, password)
  devices = [["http://fritz.box:49000", "", ""]]
  ## The http timeout to use (in seconds)
  # timeout = 5
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
  ## The cycle count, at which low-traffic stats are queried
  # full_query_cycle = 6
  ## Enable debug output
  # debug = false
```

#### Device Info (get_device_info)

#### WLAN Info (get_wlan_info)

#### WAN Info (get_wan_info)

#### DSL Info (get_dsl_info)

#### PPP Info (get_ppp_info)

### License
This project is subject to the the MIT License.
See [LICENSE](./LICENSE) information for details.