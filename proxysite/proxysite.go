// vim:set sw=2 sts=2:
package proxysite

import (
  "io/ioutil"
  "net/http"
  "net/http/cookiejar"
  neturl "net/url"
)

type ProxySite struct {
  Jar *cookiejar.Jar
}

func NewProxySite() (*ProxySite, error) {
  jar, err := cookiejar.New(nil)
  if err != nil {
    return nil, err
  }
  return &ProxySite{ Jar: jar }, nil
}

func (ps *ProxySite)Process(query string) ([]byte, error) {
  cl := &http.Client{ Jar: ps.Jar }
  resp, err := cl.Get("https://us1.proxysite.com" + query)
  if err != nil {
    return nil, err
  }
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
  resp, err := cl.PostForm("https://us1.proxysite.com/includes/process.php?action=update", data)
  if err != nil {
    return nil, err
  }
  defer resp.Body.Close()
  body, err := ioutil.ReadAll(resp.Body)
  if err != nil {
    return nil, err
  }
  return body, nil
}
