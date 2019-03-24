// vim:set sw=2 sts=2:
package proxysite

import (
  "log"
  "io/ioutil"
  "net/http"
  "net/http/cookiejar"
  neturl "net/url"
)

type ProxySite struct {
  Jar *cookiejar.Jar
  //
  Debug bool
}

func NewProxySite() (*ProxySite, error) {
  jar, err := cookiejar.New(nil)
  if err != nil {
    return nil, err
  }
  return &ProxySite{ Jar: jar, Debug: false }, nil
}

func (ps *ProxySite)dbgln(a ...interface{}) {
  if ps.Debug {
    log.Println(a...)
  }
}

func (ps *ProxySite)dbgf(f string, a ...interface{}) {
  if ps.Debug {
    log.Printf(f, a...)
  }
}

func (ps *ProxySite)Process(query string) ([]byte, error) {
  ps.dbgln(query)
  cl := &http.Client{ Jar: ps.Jar }
  req, _ := http.NewRequest("GET", "https://us1.proxysite.com" + query, nil)
  ps.dbgln(req)
  resp, err := cl.Do(req)
  if err != nil {
    return nil, err
  }
  ps.dbgln(resp)
  defer resp.Body.Close()
  body, err := ioutil.ReadAll(resp.Body)
  if err != nil {
    return nil, err
  }
  return body, nil
}

func (ps *ProxySite)Get(url string) ([]byte, error) {
  cl := &http.Client{ Jar: ps.Jar }
  data := neturl.Values{}
  data.Set("d", url)
  data.Set("allowCookies", "on")
  data.Set("server-option", "us1")
  ps.dbgln(data)
  resp, err := cl.PostForm("https://us1.proxysite.com/includes/process.php?action=update", data)
  if err != nil {
    return nil, err
  }
  ps.dbgln(ps.Jar)
  defer resp.Body.Close()
  body, err := ioutil.ReadAll(resp.Body)
  if err != nil {
    return nil, err
  }
  return body, nil
}
