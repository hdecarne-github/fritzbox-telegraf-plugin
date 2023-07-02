This [Telegraf](https://github.com/influxdata/telegraf) input plugin gathers stats from [AVM](https://avm.de/) FRITZ!Box devices. It uses the device's [TR-064](https://avm.de/service/schnittstellen/) interfaces to retrieve the stats. DSL routers as well as WLAN repeaters are supported.

This software may be modified and distributed under the terms
of the MIT license.  See the LICENSE file for details.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

### v0.1.1 (2022-03-13)
* Initial release

### v0.1.2 (2022-03-19)
* Provide pre-build archives

### v0.2.0 (2022-04-02)
* Add fritz_mesh measurement

### v0.2.2 (2023-02-19)
* Add tls_skip_verify option to ignore invalid certificates (use at your own risk)
* Update dependencies

### v0.3.0 (2023-07-02)
* Add fritz_mesh_client measurement
* Update dependencies
