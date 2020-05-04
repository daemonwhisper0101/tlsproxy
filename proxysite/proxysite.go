// vim:set sw=2 sts=2:
package proxysite

import (
  "log"
  "io/ioutil"
  "net/http"
  "net/http/cookiejar"
  neturl "net/url"
  "strings"
)

type ProxySite struct {
  Jar *cookiejar.Jar
  LastURL string
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

func (ps *ProxySite)GetWithReferer(url, referer string) ([]byte, error) {
  cl := &http.Client{ Jar: ps.Jar }

  // make referer url
  data := neturl.Values{}
  data.Set("d", referer)
  data.Set("allowCookies", "on")
  data.Set("server-option", "us1")
  ps.dbgln(data)
  resp, err := cl.PostForm("https://us1.proxysite.com/includes/process.php?action=update", data)
  if err != nil {
    return nil, err
  }
  resp.Body.Close()
  ps.dbgln(ps.Jar)
  ps.LastURL = resp.Request.URL.String()

  // make real url
  data = neturl.Values{}
  data.Set("d", url)
  data.Set("allowCookies", "on")
  data.Set("server-option", "us1")
  // no work req, err := http.NewRequest("POST", "https://us1.proxysite.com/includes/process.php?action=update", strings.NewReader(data.Encode()))
  resp, err = cl.PostForm("https://us1.proxysite.com/includes/process.php?action=update", data)
  if err != nil {
    return nil, err
  }
  resp.Body.Close()

  query := strings.TrimSuffix(resp.Request.URL.String(), "&f=norefer")
  req, err := http.NewRequest("GET", query, nil)
  req.Header.Add("Referer", ps.LastURL)
  resp, err = cl.Do(req)
  if err != nil {
    return nil, err
  }
  ps.dbgln(resp)
  body, err := ioutil.ReadAll(resp.Body)
  defer resp.Body.Close()
  if err != nil {
    return nil, err
  }
  return body, nil
}
